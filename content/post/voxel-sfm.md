---
title: "Voxel from Multicam - DIY Self Driving Part 3"
date: 2022-06-12T14:37:27-07:00
---

_This is a follow up to
[Monocular Depth Improvements]({{< ref "monocular-improvements.md" >}})
and is part of a series where I try to train models to perform common self
driving tasks from scratch._

### Background

I spent a couple of months optimizing single camera (monocular) depth models before
realizing that maybe there's a better way. One of the biggest improvements I
made to the monocular models was adding a 3D geometric constraint to enforce
that the model didn't predict depths below the ground.

Having a prior on the height of the ground cleared up most of the degenerate
cases which resulted in infinite depth predictions for things like reflective
puddles, blurry road surfaces and water on the camera.

### Problems with Image Space Depth

{{% amp-img src="/monocular-improvements/zloss-wet.png" %}}
Example of the output from a monocular depth model.
{{% /amp-img %}}

The main issue for the monocular depth models is that taking the video from multiple
cameras and merging the point clouds is hard. It's a very expensive operation
since it operates on point clouds and the result tended to be subpar.

1) The camera space depths don't have clean edges resulting in lots of extra noise at
the edges of things like cars. Filtering out that noise is expensive since it
relies on doing things like binning across frames to get a more stable output.

2) Aligning the different frames is hard. It very precise localization and
depths for the outputs to align correctly. The monocular model had scaling
issues when the car was turning since the pose network had a hard time handling
off axis movement.

3) Different camera outputs don't agree with each other since they only have one
camera as input and it can get confused at the edges.

### 3D Representation

Ideally we could have a 3D representation of the world so we don't have to solve
the 2D fusion. Luckily we already have a 3D representation--the voxel based
occupancy grids we're trying to train. We also have a model architecture that
works to convert the 2D camera feeds into that 3D representation.

{{% amp-img src="/voxel-sfm/twolane-voxel.png" %}}
Voxel representation of the world and the correspond main camera view.
{{% /amp-img %}}

In my first post I used the output from the 2D depth model to create the
training data for the 3D model. Can we cut out that transformation step and
directly learn the voxel representation?

### Differentiable Rendering: Raymarching

Differentiable rendering is a fairly new field of Machine Learning based around
training machine learning models to produce 3D representations from 2D pictures
which is exactly what we want. There's some very cool applications such as
Neural Radiance Fields. NeRFs however are primarily focused on the visual
aspects and don't map 1:1 with occupancy.

If you're interested to learn more, [PyTorch3D](https://pytorch3d.org/) is a
great library for doing things like this and my custom renderer was heavily
inspired by it.

{{% amp-img src="/voxel-sfm/raymarching.png" %}}
From "Rendering Volumes and Implicit Shapes in PyTorch3D"
https://www.youtube.com/watch?v=g50RiDnfIfY&t=501s
{{% /amp-img %}}

We're focusing on Raymarching since it's a technique to render 3D voxel
representations. Most differentiable rendering is focused on reproducing a visual model of the
world that matches the input images. Typical raymarching has two parts:

1) a grid of probabilities
2) the colors for each voxel in the grid

To render these into an image, we can cast rays into the 3D representation and
at each point along the ray grab the probability and the color.

For rendering we just want the first color and not all of them so we can do a
cumulative operation on the probabilities to just take the ones in front and not
behind (see the diagram above).

Once we have the probabilities we can multiply the colors by them and do a sum.
This results in a 2D image and since we use a differentiable framework (PyTorch)
we can compute the gradient on it and do stochastic gradient descent on it like
any other machine learning model.

### Rendering Depth

While differentiable rendering already exists, I haven't seen anyone use it for
doing structure from motion in a multicam environment.

In our case we don't care about the colors and instead just want depth. Thus we
can multiply the probabilities by the distances from the camera.

```python
def render_dist(self, grid, K, ex, CAST_D):
    """
    Renders depths from a voxel representation.

    grid: the 3D occupancy grid
    K: the camera intrinsics matrix (focal distance etc)
    ex: the 3D matrix transform for the camera position and rotation
    CAST_D: the length of the ray
    """

    # The sampling coordinates and distances are fixed to the camera so we don't
    # need to compute the gradient and can save some memory.
    with torch.no_grad():
        # generate the XYZ coordinates for each pixel ray
	points, point_dists = self._points(K, ex, CAST_D)
	# generate all of the distances for each square in the voxel grid
	dist = self._dist_grid(ex)

    # sample the grid probabilities and distances
    sampled = F.grid_sample(torch.cat((dist, grid), dim=1), points)
    sampled_dist = sampled[:, 0]
    sampled = sampled[:, 1]

    # set the furthest point in the ray to be 100% so the cumsum always equals 1
    sampled[:, :, :, CAST_D-1] = 1
    sampled_dist[:, :, :, CAST_D-1] = CAST_D/scale

    # mark all points below the ground as solid
    subzero = points[:, :, :, :, 0] < -1
    sampled[subzero] = 1
    sampled_dist[subzero] = point_dists.expand(sampled_dist.shape)[subzero]

    # compute the ray weights by applying a cumlative sum, clamping to 1 and
    # then computing the differences
    probs = sampled.cumsum_(dim=3)
    probs = probs.clamp_(max=1)
    probs = probs.diff(dim=3, prepend=torch.zeros((BS, img_h, img_w, 1))

    # multiply the probabilities by the distances to get the final depth
    return (probs*sampled_dist).sum(dim=3)
```

I'm using a slightly atypical formula with cumsum when rendering instead of the
emission formula in PyTorch3D shown above.
Using `.cumsum().clamp_(max=1).diff()` to compute the probabilities seems to
work well for this problem and is guaranteed to sum to 1 so there's no extra
scaling needed.

With this we can render a 2D depth map from the 3D representation directly.

{{% amp-img src="/voxel-sfm/rightpillar.png" %}}
A picture from the right pillar camera and the 2D depth map rendered from the
occupancy grid. The bottom is striped since we clamp the distance for points
underground.
{{% /amp-img %}}

### Training

Now that we have 2D depth maps we can apply the same training techniques as
using in a monocular depth model. We can use pairs of frames and the speed of
the vehicle to compute the loss on the generated depth map. This uses the exact
same structured similarity loss as
[monodepth2](https://github.com/nianticlabs/monodepth2) though I ditched the
smooth loss and the automasking.

For the pose network I've augmented it to take in the IMU details and the speed
of the vehicle but applied the same distance loss to keep is scaled to real life
as in my first post, given the geometric constraints I'm not sure it's strictly
necessary but helps during early stages of training.

{{% amp-img src="/voxel-sfm/gridz.png" %}}
The occupancy grid probabilities from the top down and the positions of the
vehicle during training. Vehicle moves from the top down.
{{% /amp-img %}}

I compute the loss for each camera at 2 different vehicle locations. The first
location is always the same as the data fed into the model and the second is
randomly selected from the next 72 meters. Using multiple locations greatly
improves the output since it is accurate from multiple viewpoints and gives
extra feedback for points far in the future.

There's a bit of artifacting from the first camera as the model cheats a bit
with the distance but none from the second since it's randomly positioned. An
improvement would be to randomize the position of the first camera too.

I ended up rendering the depth at 320x240 and computing the loss at the same
resolution too. This is half of the resolution of the 2D representation but
seems to be accurate nonetheless.

I'm also now using all of the side/forward facing cameras since adding in narrow
and fisheye improves the long accuracy and fills in the gaps between the main
and pillar cameras.


### Failed Attempt: Directly Learn the Colors + Occupancy

I also tried to learn the colors and occupancy directly so I didn't have to use
the SfM disparity loss but that didn't work as well.

{{% amp-img src="/voxel-sfm/colorrender.png" %}}
The target, the rendered model output and the occupancy grid.
{{% /amp-img %}}

The model ended up cheating and created a colored bubble instead of actually
learning the correct occupancy. This was only trained from a single position
instead of multiple and I later found a bug causing the aspect ratio to be
messed up. With those improvements this might be a viable approach to revisit.

### VoxelNet v2

One of the biggest limitations of my original voxel transformer model was the
dimensionality, the grid was only 144x80 which with a 1/3 meter scale meant 48m
x 27m which just isn't enough.

The transformer was structured to have one transformer with one
query entry per X/Y coordinate. Combined with the high number of keys from the
per camera features this got very expensive to create a larger grid. I spent
some time iterating on it and ended up with a smaller transformer with a larger
value dimension and a standard convolution upsampling afterwards to reach the
target size of 256x256 which is ~6x bigger output than my original model in a
similar amount of memory.

{{% amp-img src="/voxel-sfm/voxelnet_v2.png" %}}
The VoxelNet v2 model.
{{% /amp-img %}}

In hindsight my original model was doing feature fusion way too early. I was
concatenating the per camera inputs and running a joint BiFPN trunk on the
inputs before feeding it into a single transformer. Before the transformer the
features are all in image space so feeding them jointly through a BiFPN was
useless since they weren't in the same space.

Thanks to the authors who wrote the papers
[BevFormer](https://arxiv.org/pdf/2203.17270.pdf) and
[BevSegFormer](https://arxiv.org/pdf/2203.04050.pdf) which inspired the
upsampling and per camera heads.

### Results

I'm quite happy with the results. The fused output is much nicer than my
previous attempts to fuse the point clouds. It's outputting much further away
and the output is much more useful.

{{% amp-img src="/voxel-sfm/voxel-highway-cars.png" %}}
A voxel representation of a highway.
{{% /amp-img %}}

There is some artifacting around the vehicle due to the fixed camera positions
but overall it's quite good. The barrier/hill on the right is present all the
way along the road for the full 85m and it's straight which matches reality
quite well.

Notably all of the dynamic objects/vehicles are gone from the output. Since this
is trained from multiple different positions and the cars move they're entirely
omitted from the output unless it's very clear that they're parked.

{{% amp-img src="/voxel-sfm/voxel-building-corner.png" %}}
Approaching an intersection with a building blocking the view to the right.
{{% /amp-img %}}

This is the vehicle approaching an intersection. The model correctly understands
that there's open space past the building despite not seeing it and the road
slants up to the right. It also captures the nearby plants though with low
probability since they're very close to the z cutoff point.

{{% amp-img src="/voxel-sfm/voxel-bridge.png" %}}
Going over a short bridge with trees nearby. 90% confidence top, 50% confidence
bottom
{{% /amp-img %}}

The bridge railing is accurately captured with a gap behind it. Confidence is
high for much of the solid objects as the 90% confidence threshold captures most
of the important areas. The overhanging tree boundaries match the actual space
and is completely solid which indicates the model isn't just learning shells.

Tesla's voxel output only outputs the shells of the object around them which in
contrast to these results seems to imply that Tesla is training their voxel
models in a supervised manner from point cloud representations.

{{% amp-img src="/voxel-sfm/voxel-obstructed.png" %}}
Water on right pillar camera obstructing the lens.
{{% /amp-img %}}

With a partially obstructed camera we still get fairly reasonable results. The
main camera can see the building ahead and the part of the building that is
visible is present. There is some output gap where the camera is obstructed and
no other cameras can see that area.

{{% amp-img src="/voxel-sfm/voxel-narrow.png" %}}
Driving between two buildings close together.
{{% /amp-img %}}

The building is shown correctly off axis and is consistent between camera
transitions. The small pillars and posts aren't captured by the model possibly due
to them being smaller than the 1/3m resolution. The arch over the road
isn't represented but the roofs are possibly due to the 4m height cap.

### Conclusion

While these results are far from perfect it's much better than what I had
before. Plus, it's way simpler to train the voxel solution (and faster!) since
it's all done E2E. With the geometric constraint of the grid it seems that many
of the degenerate cases affecting monocular depth cameras (infinite depth,
reflections, obstructions, localization) are mitigated and the training is much
more robust to them.

This training setup seems conceptually similar to how
[DETR3D](https://arxiv.org/abs/2110.06922) operates for doing bounding boxes of
3D objects. From my understanding, both this and DETR3D learn a 3D
representation directly from the 2D camera space outputs. For doing
object detection instead of using depth generated maps you could render 3D
bounding boxes and use image space labels. Similarly for road lines you could
render the road classes and use image space lane lines.

While my approach has analogs to SOTA methods in other domains, I believe this
is a novel approach in the structure from motion problem space. Very happy to be
able to share new learnings from this project.

### Citing

If you're interested in writing a paper on this feel free to cite me. I'd also
be happy to chat / collaborate. I thought about writing a paper but I'd have to
redo all all the training on nuScenes instead of my own dataset.

### Thanks

Thanks to Jeremy from the PyTorch3D project for helping me understand
differentiable rendering and making this possible.
