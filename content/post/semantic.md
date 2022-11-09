---
title: "3D Semantic Segmentation - DIY Self Driving Part 4"
date: 2022-11-08T20:28:04-08:00
---

_This is a follow up to
[Voxel from Multicam]({{< ref "voxel-sfm.md" >}})
and is part of a series where I try to train models to perform common self
driving tasks from scratch._

I've previously put together occupancy models for self driving but that's only
one specific perception task.

Another common driving task is semantic segmentation. Semantic segmentation
takes in the image and for every pixel predicts a specific class. This can be
used to tell things like walls apart from cars or classify different types of
lane lines and curbs on a road.

Classical segmentation operates on images but since we're operating in 3D it'd
be nice to get the same classes as either a birdseye view representation or a
voxel representation.

### Supervised vs Unsupervised Learning

The occupancy grid models that I've made before are trained in an unsupervised
manner. This means that there's generated "ground truth" to train the model on
and it instead uses consistency between frames to learn the occupancy grid. This
makes it simpler since I don't need to collect any labels which can be very
expensive.

Segmentation is a supervised task. It's trained by having a ground truth (often
human labeled) to compare the model output to. Luckily for us, there's a number
of publicly available datasets that I can pull from.

One dataset that matches our tasks fairly well is
[BDD100K](https://www.bdd100k.com/)
which has 100k labeled images for things like lane lines and drivable space. For
full semantic segmentation it only has about 10k images but should be enough for
our purposes.

### Auto-labeling Models

How do we take the BDD100K dataset and apply it to a completely different
driving dataset to learn a 3D representation?

We can use a pretrained model as an "auto-labeler". This second model is trained
on BDD100K and we can run it on our dataset to generate ground truth labels for
our model.

{{% amp-img src="/semantic/main-semantic.png" %}}
Main camera and the semantic segmentation output using UPerNet + ConvNeXt-T
trained on BDD100K.
{{% /amp-img %}}

{{% amp-img src="/semantic/bdd100k-road.png" %}}
Sample from BDD100K with predictions from my fine tuned YOLOP model predicting
lane classes and drivable space probabilities.
{{% /amp-img %}}

I used a retrained [YOLOP](https://pytorch.org/hub/hustvl_yolop/) for the road surface
segmentation
and for the semantic segmentation I used
[a pretrained UPerNet with a ConvNeXt-T backbone](https://github.com/SysCV/bdd100k-models/tree/main/sem_seg#convnext).

Using pretrained models on similar tasks made it a lot easier for me to quickly
fine-tune the models and start training the 3D model.

Thanks to the folks who trained and shared those models. They made training
these models way easier. Citations:

```latex
@misc{2108.11250,
    Author = {Dong Wu and Manwen Liao and Weitian Zhang and Xinggang Wang},
    Title = {YOLOP: You Only Look Once for Panoptic Driving Perception},
    Year = {2021},
    Eprint = {arXiv:2108.11250},
}
@InProceedings{bdd100k,
    author = {Yu, Fisher and Chen, Haofeng and Wang, Xin and Xian, Wenqi and Chen,
              Yingying and Liu, Fangchen and Madhavan, Vashisht and Darrell, Trevor},
    title = {BDD100K: A Diverse Driving Dataset for Heterogeneous Multitask Learning},
    booktitle = {IEEE/CVF Conference on Computer Vision and Pattern Recognition (CVPR)},
    month = {June},
    year = {2020}
}
```

and thanks to [Thomas E. Huang](https://github.com/thomasehuang) for training
the bdd100k segmentation model.


### Fine Tuning Datasets

BDD100K only has forwards facing dashcam footage as part of it's training set so
it works well for the forward facing camera in the vehicle but less well for
the other cameras such as the backwards facing side repeaters. To augment this
dataset we can hand label some data and use it to retrain the model.

In this case I spent a couple of days labeling about 300 images. While 300
images is much less than 100k we can do some tricks to weigh those examples more
during training in order to make the model prioritize our examples.

{{% amp-img src="/semantic/label-studio.png" %}}
Labeling road lines and curbs in Label Studio.
{{% /amp-img %}}

For the labeling software I used
[Label Studio](https://github.com/heartexlabs/label-studio)
since it's open source and fairly easy to get setup with. There's definitely
some rough edges but it has some nifty features. I setup autolabeling within
Label Studio so I would only have to fine tune the results from the model
instead of labeling the entire image myself.

### Labelling Inference

Since these models are just being used as training data we can cut their
computational usage by running them in inference mode and in lower precision.

```python
model = load_semantic_model()
# switch to eval mode
model = model.eval()
# convert the model to fp16 for performance boost and to take up less memory
model = model.half()

# run the model in inference mode to disable autograd tracking
with torch.inference_mode():
    target = model(input)
```

You likely could quantize these to int8 for even faster inference and less
memory usage but I didn't bother.

### Voxel Segmentation Model

The previous occupancy model outputted a single probability for each of the
points in the voxel grid. For segmentation we need to add in probabilities for
each of the various classes. We can do this by adding an extra head to our model
decoder. Now we have two final layers, one to predict occupancy probabilities
and one to predict the semantic classes for each voxel.

{{% amp-img src="/semantic/voxel-semantic.png" %}}
Example output from the model. In clock wise order from top left:
main camera,
losses on classes,
BEV occupancy, BEV semantic probabilities,
argmax of target semantic classes, argmax of predicted classes.
The ground and sky classes have been intentionally omitted so the argmax for
those areas is noisy.
{{% /amp-img %}}

To convert this occupancy + semantic classes into an image that we can compare
to the autolabeling model we use the same differentiable rendering technique
that I used for generating the depth maps. Instead of computing the depths it
instead renders out the class probabilities.

There's two ways of comparing these losses:

1. Threshold the target probabilities and use binary cross entropy loss -- this
    allows weighing the different classes for low frequency classes
2. Use MSE on the raw -- presigmoid outputs from the model. This allows the new
   model to learn the probabilities. This loss is very similar to a
   teacher-student model distillation.

Thresholding seemed to work well for lane lines since I could weigh the positive
lane class higher since there's a class imbalance. There's likely a way to weigh
positive classes more with the raw outputs but I haven't investigated this
further.

For rendering I used a custom DepthEmissionRaymarcher with PyTorch3D. This
allows for rendering the depth as well as the semantic labels in a single pass.

```python
class DepthEmissionRaymarcher(torch.nn.Module):
    def __init__(self, background: Optional[torch.Tensor] = None) -> None:
        super().__init__()

        self.floor: float = 0
        self.background = background

    def forward(
        self,
        rays_densities: torch.Tensor,
        rays_features: torch.Tensor,
        ray_bundle: "RayBundle",
        eps: float = 1e-10,
        **kwargs,
    ) -> torch.Tensor:
        """
        Args:
            rays_densities: Per-ray density values represented with a tensor
                of shape `(..., n_points_per_ray, 1)` whose values range in [0, 1].
            rays_features: Per-ray feature values represented with a tensor
                of shape `(..., n_points_per_ray, feature_dim)`.
            eps: A lower bound added to `rays_densities` before computing
                the absorption function (cumprod of `1-rays_densities` along
                each ray). This prevents the cumprod to yield exact 0
                which would inhibit any gradient-based learning.
        Returns:
	    * depth: the depths at each ray
	    * features: the features at each ray
        """
        device = rays_densities.device

        rays_densities = rays_densities.clone()

        # clamp furthest point to prob 1
        rays_densities[..., -1, 0] = 1
        # set last point to background color
        if self.background is not None:
            rays_features[..., -1] = self.background

        # set floor depths
        # depth = (z-z0)/vz
        floor_depth = (
            (self.floor - ray_bundle.origins[..., 2]) / ray_bundle.directions[..., 2]
        )
        torch.nan_to_num_(floor_depth, nan=-1.0)
        floor_depth[floor_depth <= 0] = 10000
        floor_depth = floor_depth.unsqueeze(3).unsqueeze(4)
        rays_densities[ray_bundle.lengths.unsqueeze(4) > floor_depth] = 1

        ray_shape = rays_densities.shape[:-2]
        probs = rays_densities[..., 0].cumsum_(dim=3)
        probs = probs.clamp_(max=1)
        probs = probs.diff(
            dim=3, prepend=torch.zeros((*ray_shape, 1), device=device)
        )

        depth = (probs * ray_bundle.lengths).sum(dim=3)
        features = (probs.unsqueeze(-1) * rays_features).sum(dim=3)

        return depth, features
```

### Mesh Segmentation Model

For the road surfaces I also experimented with an alternative 3D
representation--a mesh. The mesh is built from a height map and a probability
map for each X/Y location. This is then rendered using PyTorch3D back into image
space to apply the SfM and semantic losses.

{{% amp-img src="/semantic/mesh-semantic.png" %}}
Example output from the model. In clock wise order from top left:
main camera,
predicted depth,
BEV drivable space probabilities, BEV lane line probabilities,
predicted lane lines and drivable space, the target lane lines and drivable
space from autolabeling model.
{{% /amp-img %}}

Similar to the voxel model, the areas far from the camera that aren't directly seen by the cameras end
up having random values since there's no constraints on them.

I contributed depth shaders to PyTorch3D that enable rendering depth maps of the
meshes. This enables doing a joint SfM loss and the segmentation loss.

https://github.com/facebookresearch/pytorch3d/blob/main/pytorch3d/renderer/mesh/shader.py#L400

To minimize computation I first render the fragments and then the individual
shaders so I don't have to rerender for each output:

```python
sigma = 1e-4 / 3
raster_settings = RasterizationSettings(
    image_size=(240, 320),
    faces_per_pixel=10,
    blur_radius=np.log(1.0 / 1e-4 - 1.0) * sigma,
)
self.rasterizer = MeshRasterizer(
    raster_settings=raster_settings,
)
self.shader_depth = SoftDepthShader(
    device=device,
    cameras=cameras,
)

# Uses a HardFlatShader with a white AmbientLight for rendering the semantic
# classes with no modifications.
self.shader_class = HardFlatShader(
    device=device,
    cameras=cameras,
    lights=AmbientLights(device=device),
)

## inference
cameras = CustomPerspectiveCameras(
    T=T,
    K=K,
    image_size=torch.tensor(
	[[h // 2, w // 2]], device=device, dtype=torch.float
    ).expand(BS, -1),
    device=device,
)
render_args = dict(
    cameras=cameras,
    zfar=100.0,
    znear=0.2,  # empirically from voxel we don't see closer than 1.3
    # cull backfaces to prevent the ground becoming the ceiling
    cull_backfaces=True,
    eps=1e-8,  # minimum scaling factor for transform_points to avoid NaNs
)
# generate the fragments for the shaders to shade
fragments = self.rasterizer(meshes, **render_args)

# shade depth map
depth = self.shader_depth(fragments, meshes, **render_args)[..., 0]

# shade semantic classes
classes = self.shader_class(
    fragments, meshes, **render_args
).permute(0, 3, 1, 2)
```

Training a mesh height map sort of worked but had some issues. Since the height
map is usually perpendicular to the camera it's hard to learn the heights. When
the height is too low it's not rendering over the target pixel so there's no
gradient for it to learn from.

### Autograd Graph Breaks: 50% Memory Savings!

Rendering all these different camera views and autolabeling models is very
memory intensive. Since all of the individual camera losses are independent you
can cut down on the memory usage by pre-emptively computing the gradients by
calling `.backward()` on each independent loss.


{{% amp-img src="/semantic/autograd-pause.png" %}}
Proactively calling `.backward()` and pausing PyTorch autograd computation to
save compute.
{{% /amp-img %}}

To avoid running the full backwards pass I came up with some handy helpers to
make it easy to pause the graph and accumulate the gradient to an intermediate
tensor -- in this case the voxel/mesh output.

```python
def autograd_pause(tensor):
    """
    autograd_pause returns a new tensor with requires_grad set. The original
    tensor is available as the .parent attribute.

    See autograd_resume.
    """
    detatched = tensor.detach()
    detatched.requires_grad = True
    detatched.parent = tensor
    return detatched

def autograd_resume(*tensors):
    """
    autograd_resume resumes the backwards pass for the provided tensors that were
    graph broken via autograd_pause.
    """
    torch.autograd.backward(
        tensors=[t.parent for t in tensors],
        grad_tensors=[t.grad for t in tensors],
    )
```

To use it you can do something like:

```python
grid = model(input)

# accumulate gradients into grid.grad
grid = autograd_pause(grid)

for camera in cameras:
    # do the rendering
    loss = ...
    loss.backward()

# complete the backwards pass for the model
autograd_resume(grid)
```

### FlashAttention

[FlashAttention](https://github.com/HazyResearch/flash-attention) is a hip new
way of doing multiheaded attention in PyTorch. I decided to port my model over
to use BF16 for memory+performance reasons and also switched my model to use
flash attention instead of the default PyTorch implementation.

This cut my memory usage by ~8% (17522MiB vs 18950MiB) but since the transformer
is only a part of the E2E model training it likely had less benefit than on a
heavier text transformer model.

The API is very much intended for text style models with varying sequence
lengths but it's straight forward to pass in a fixed grid for cross attention.

```python
from flash_attn.flash_attn_interface import (
    flash_attn_unpadded_kvpacked_func,
)

...

# (BS, output height, output width, transformer embedding dim)
query = self.query_encoder(context)
query = query.reshape(BS, -1, TRANSFORMER_DIM)
q_seqlen = query.shape[1] # number of output positions per batch

# (BS, image height, image width, 2, transformer embedding dim)
kv = self.kv_encoder(x)
kv = kv.reshape(BS, -1, 2, TRANSFORMER_DIM)
k_seqlen = kv.shape[1] # number of keys per batch

bev = flash_attn_unpadded_kvpacked_func(
    # reshape into (# keys, HEADS, per head dim)
    q=query.reshape(-1, HEADS, TRANSFORMER_DIM//HEADS),
    # reshape into (# keys, 2, HEADS, per head dim)
    kv=kv.reshape(-1, 2, HEADS, TRANSFORMER_DIM//HEADS),
    cu_seqlens_q=torch.arange(0, (BS+1)*q_seqlen, step=q_seqlen, device=query.device, dtype=torch.int32),
    cu_seqlens_k=torch.arange(0, (BS+1)*k_seqlen, step=k_seqlen, device=query.device, dtype=torch.int32),
    max_seqlen_q=q_seqlen,
    max_seqlen_k=k_seqlen,
    dropout_p=0.0,
)

bev = bev.reshape(BS, self.D, self.W, TRANSFORMER_VALUE_DIM)
bev = bev.permute(0, 3, 1, 2)
# output (BS, channels, height, width)
```

### TVL1 Loss

I've also added a
[total variation L1 loss](https://github.com/facebookresearch/neuralvolumes/blame/main/models/decoders/voxel1.py#L224-L229)
from the
[Neural Volumes paper](https://research.facebook.com/publications/neural-volumes-learning-dynamic-renderable-volumes-from-images/).

This helps reduce per camera artifacts as well as reduces the amount of random
noise in the grid outside of the rendered camera view.
