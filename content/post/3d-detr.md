---
title: "3D Dynamic Objects - DIY Self Driving Part 5"
date: 2023-01-21T13:51:43-08:00
---

_This is a follow up to
[3D Semantic Segmentation]({{< ref "semantic.md" >}})
and is part of a series where I try to train models to perform common self
driving tasks from scratch._

I decided to switch areas of focus for this new model. Previously I had been
working entirely with dense models which output dense representations about the
world such as the voxel occupancy grids and the BEV semantic maps for lane lines
and drivable space.

One of the areas I hadn't tried to solve much was dynamic objects. My previous
models were heavily dependent on structure from motion losses as well as an
assumption the world was static so the car could move through the representation
and compute losses at multiple points. Things like cars, bikes, people, etc
they move and violate that assumption.

To model moving objects you need 1) a representation that can understanding
moving objects and 2) a model that can consume multiple frames to track objects
either through some historical state or by being able to process batches of
frames.

### DE⫶TR: End-to-End Object Detection with Transformers

[DETR (by Nicolas Carion, Francisco Massa, et. al)](https://arxiv.org/abs/2005.12872)
uses transformers for 2d object detection by directly outputting 2d bounding boxes and their classes.
The model uses a set of possible detections with a learned set of transformer
keys for cross attention. Each one of these keys queries the model output to
produce a set of one hot encoded class probabilities as well as the actual XY
coordinates and height width of the detection.

{{% amp-img src="/3d-detr/detr-detailed.png" %}}
The DETR architecture. The decoder uses learned object queries to directly
generate the bounding box predictions.
{{% /amp-img %}}

Since the decoder is transformer based it fits well into the existing
transformer based models I've been using. The main contribution here is the
learned queries but the k/v pairs can be basically anything. The output can be
easily extended into 3D to predict XYZ coordinates, sizes, classes, rotations
and velocities.

### Reference: Single Shot Detector (SSD)

There's a number of other detection models but they often require multiple
stages i.e. region proposals in RCNN or post processing i.e. non maximal
suppression (NMS).

{{% amp-img src="/3d-detr/ssd.png" %}}
[Single Shot Detector (SSD)](https://arxiv.org/pdf/1512.02325.pdf) outputs. Each
ground truth box may have multiple overlapping predicted boxes since SSD
predicts the offset and size for any overlapping objects.
{{% /amp-img %}}

These object detection models can be extended to 3D but they tend to output
multiple possible boxes and during training optimize all nearby candidate
regions. SSD can be converted into a BEV based model by having a grid of
predictions on the X/Y plane and then for each square predicting the class as
well as size and offsets of any objects in them. This requires having a 3D
ground truth to train.

### Related Work

There's been a few attempts at doing a 3D DETR styled model:

* [3DETR](https://arxiv.org/pdf/2109.08141.pdf) predicts 3D bounding boxes from
    pointclouds using a stack of transformers with object queries to output 3D
    boxes.
* [DETR3D](https://arxiv.org/pdf/2110.06922.pdf) predicts 3D bounding boxes from
    multiple cameras using an CNN encoder per camera and then a stack of
    transformers with object queries to output 3D boxes.

Both of these models are trained with known ground-truth 3D bounding boxes.
DETR3D is most similar with my approach but there's major differences in both
the encoder and training strategy.

### Temporal BEV

To handle moving objects we need to be able to consume multiple frames. To do
this I've decided to extend the existing architecture to instead generate an
intermediate birdseye view feature map. The output is a 2D grid of features
roughly corresponding to X/Y coordinates around the vehicle.

{{% amp-img src="/3d-detr/bev-features-transformer.png" %}}
A generalized version of my previous VoxelNet models that outputs a birdseye
feature map for a downstream model to consume.
{{% /amp-img %}}

The generic version of the VoxelNet model allows you to attach any transformer
decoder you want. We could use this same intermediate features for doing voxel
representations, lane lines and detections. Training a joint model becomes very
tractable.

To handle temporal features we can simply stack these per frame BEV feature maps
and use a CNN to be able to learn changes across frames for things like dynamic
objects.

{{% amp-img src="/3d-detr/bev-temporal.png" %}}
A simple mixing CNN-based network to learn temporal changes across multiple
frames.
{{% /amp-img %}}

I'm just using the current frame and 2 past frames (3 total) but it should be
straight forward to extend it to more. The same encoder network is run on each
frame and the idea is to capture all static information on a per frame basis.
Assuming the encoding network correctly learns the positions of objects in an XY
grid, a CNN can use a convolution to learn the relative velocities by diffing
across the frames.

The output from this model is just another BEV feature map which can be used by
any of the existing 3D output heads. This makes adding temporal learning "plug
and play" across my existing tasks though the loss function will need to be
temporal aware as well for things like voxel outputs.

### BEV DETR Head

We can use a DETR style decoder and use it with the BEV feature map. In the
multiheaded cross attention the Q is the DETR object queries, the K is the BEV
positional encoding and then the V is the features.

{{% amp-img src="/3d-detr/bev-detr-head.png" %}}
The DETR BEV head. Consumes the temporal BEV map and outputs a fixed set of
predictions including 3D position, velocity, dimensions and class.
{{% /amp-img %}}

The class is encoded using a one hot encoding and all of the other fields are
using sigmoid multiplied by a fixed range to get to concrete distances, sizes
and velocities.

### Training in Image Space

Since I'm training all of my models on my own personal data -- I don't have
ground truths for any of these object detections. I could pay people to label
them but that's time consuming and expensive for an individual. I decided to
continue using my previous approach of using a image space model to generate the
targets that are used to train the model. This means I have to depend on single
camera/image space models and can't use 3D ground truths that most of these
papers depend on.

I used a pretrained CascadeRNN model with a ConvNeXt-T backbone that's provided
by the
[bdd100k-models project](https://github.com/SysCV/bdd100k-models/tree/main/det)
which is trained on BDD100K which I've found matches quite well with the driving
footage from my car.

{{% amp-img src="/3d-detr/bdd100koutput.png" %}}
An example image from the main camera with predictions from a pretrained image
space detection model.
{{% /amp-img %}}

My model outputs 3D bounding boxes though so I need a way to convert those into
image space so I can use the image space loss on them. To do this I convert each
cuboid into 8 points one for each corner, project them into each camera and then
take the max/min for each detection to generate an image space bounding box. Max
and min generally aren't differentiable but since we just care about getting the
outline to match it's sufficient for training purposes.

Here's the code to convert outputs into image space boxes for a particular
camera.

```python
def points_to_bboxes2d(
        points: torch.Tensor, K: torch.Tensor, ex: torch.Tensor,
        w: int, h: int) -> torch.Tensor:
    """
    points_to_bboxes2d projects the 3d bounding boxes into image space
    coordinates.
    Args:
        points: (BS, num_queries, 8, 3)
        K:  camera intrinsics (world to camera)
        ex: camera extrinsics (camera pos to world)
        w: image width
        h: image height
    Returns:
        pix_points: (BS, num_queries, 8, 2)
        bboxes: (BS, num_queries, 4)
    """

    BS = len(K)
    num_queries = points.shape[1]
    device = K.device

    K = K.clone()
    K[:, 0] *= w
    K[:, 1] *= h
    K[:, 2, 2] = 1

    # convert to list of points
    points = points.reshape(-1, 3)
    ones = torch.ones(*points.shape[:-1], 1, device=device)
    points = torch.cat([points, ones], dim=-1).unsqueeze(2)

    inv_ex = ex.pinverse()
    # inv_ex: convert to image space
    # K: convert to local space
    P = torch.matmul(K, inv_ex)

    # repeat for each query*points combo
    P = P.repeat_interleave(num_queries*8, dim=0)

    points = torch.matmul(P, points)

    # identify boxes that are behind the camera to avoid matching
    invalid_mask = (points[:, 2, 0] < 0).reshape(BS, num_queries, 8).any(dim=2)

    pix_coords = points[:, (0,1), 0] / (
        points[:, 2:3, 0] + 1e-8
    )
    pix_coords = pix_coords.reshape(BS, num_queries, 8, 2)

    xmin = pix_coords[..., 0].amin(dim=-1)
    xmax = pix_coords[..., 0].amax(dim=-1)
    ymin = pix_coords[..., 1].amin(dim=-1)
    ymax = pix_coords[..., 1].amax(dim=-1)

    bbox = torch.stack((xmin, ymin, xmax, ymax), dim=-1)

    return pix_coords, bbox, invalid_mask
```


### Matching Predictions to Targets

DETR relies on using Hungarian matching to match the predictions to the targets.
I've adapted this to do the matching for each camera for each frame to learn the
bounding box positions. Instead of penalizing unmatched boxes on each image, I
instead identify boxes not matched by any camera and apply loss to the unmatched
class.

{{% amp-img src="/3d-detr/num-matched.png" %}}
Number of predicted vs matched boxes. I can adjust the loss weight to calibrate
these during training.
{{% /amp-img %}}

One issue with the 3D to 2D conversion is that boxes behind the camera can get
incorrectly projected into image space. I've added an extra cost for invalid
boxes to avoid them from being matched by the matcher. A better option would be
to do the matching using polar coordinates which would allow the prediction to
rotate to be in view if it gets matched.

```python
# Add cost for invalid boxes
cost_invalid = torch.zeros_like(cost_bbox)
if invalid_mask is not None:
    cost_invalid[invalid_mask.reshape(-1), :] = self.cost_invalid

# Final cost matrix
C = (
    self.cost_bbox * cost_bbox
    + self.cost_class * cost_class
    + self.cost_giou * cost_giou
    + cost_invalid
)
```

### Results

There's two things to look at for this 1) is the image space/loss performance
and 2) is the actual 3D representation and accuracy.

{{% amp-img src="/3d-detr/imagespace-examples.png" %}}
Example predictions from the BEV DETR model from the main camera.
{{% /amp-img %}}

The outputted image space predictions are quite accurate. The bounding boxes fit
quite well and are generally no more than slightly off. Different classes of
objects such as signs vs cars are accurately outputted.

{{% amp-img src="/3d-detr/outputs.gif" %}}
Bounding boxes with velocity across multiple frames for the narrow forward
camera.
{{% /amp-img %}}

The velocity output is far from perfect. Using only 3 frames as input and 3
frames to train on does have it learn some of the motion of things like the
vehicle in front but for static signs etc it doesn't track that well. Increasing
the time window/spacing the frames out + training it for longer likely would
help improve the accuracy there. At 36 fps, there's only 0.0278 seconds between
each frame which is extremely short but does let the model learn a bit.


{{% amp-img src="/3d-detr/combined-0.png" %}}
The same scene rendered as 3D bounding boxes in THREE.js.
{{% /amp-img %}}

When we look at the model output in 3D we see that while all objects are
detected, the predicted locations/sizes aren't all that accurate. Nearby objects
tend to be more accurate but for objects that are far away there's quite a bit
of size confusion. Moving an object further away vs making it smaller is
equivalent in image space so for far away objects that are "static" across the
multiple frames can end up as tiny boxes. We also see that some of the objects
in the back left/right have duplicate small boxes with one for each of the
cameras.

Some possibilities for improving this could be:

* Increasing the time duration between frames/add more frames to make objects
* move further to get better estimate on distance.
* Add a size prior for different classes of objects.
* Add a regularization loss on the number of boxes to try and avoid duplicate
boxes across different cameras.
* Add a model to match objects across two cameras.
* Use a ground truth such as LIDAR/Radar to eliminate the size/distance confusion.
* Adjust Hungarian matcher to prefer boxes matched by other cameras.


{{% amp-img src="/3d-detr/complex-scene.png" %}}
A birdseye view of a very complex scene with numerous cars, pedestrians, traffic lights and signs.
{{% /amp-img %}}

For very complex scenes it does a good job of recalling all objects. There's a
handful that are missed but this may be because the model only has 100 object
slots and this scene has 69 detections. Typically DETR models require many more
slots than predictions.

{{% amp-img src="/3d-detr/complex-vertical.png" %}}
A lateral view of the intersection.
{{% /amp-img %}}

The elevation of objects seems fairly accurate, the stop lights and street signs
appear to be at a correct elevation above the vehicle. Pedestrians and cars are
on the ground.

{{% amp-img src="/3d-detr/complex-leftpillar.gif" %}}
3 frames with predictions from the left pillar camera of an intersection.
{{% /amp-img %}}

The side cameras are a bit more noisy, partially due to the pretrained BDD100K
model which only is trained on forward facing views but also due to the fact
that there's less overlap with other cameras. The vanishing objects are
predicted once for all frames but the BDD100K model is noisy which is causing
them to disappear due to no match.

### Conclusion

I learned a ton about temporal models and object detection models over the last
couple of months. Sparse detections are quite different from my previous dense
models and adding in a RNN aspect of the models made added a lot of complexity
to training and data loading. This requires a lot more frames per example than
my previous models and the CPU decoding/preprocessing is now the current
bottleneck from me being able to train on more than 3 input frames and 3 output
frames.

Now that I have successfully being able to train models on dynamic objects it
would be fun to go back to the voxel networks and try and train them with a
temporal loss across frames to be able to capture dynamic objects in the grids.

Multitask training across voxel, dynamic objects and semantic BEV maps would be
a very interesting option since ideally it would result in a highly accurate
intermediate representation that you could potentially use as a backbone for
many different driving tasks.
