---
title: "DIY Self Driving Part 2: Monocular Improvements"
date: 2022-06-12T14:20:04-07:00
---

_This is a follow up to [DIY Self Driving]({{< ref "diy-self-driving.md" >}})._

In the past few months I've been iterating on my previous work on creating self
driving models. The main goals were initially:

1. train depth models for each camera
2. generate joint point clouds from the multiple cameras
3. use the fused outputs to create a high quality reconstruction that I can use
   to label things like lane lines

This post lists all the various problems I ran into and some of the mitigations
I applied for those issues.

### Off-axis Motion

For the side cameras the easiest thing to do is to apply the same monodepth2
techniques that I had used for the main camera and train a model on them. I ran
into a lot of issues since the side cameras especially at higher speed can have
very large differences and there's few good features between the two frames. The
off-axis motion resulted in highly inaccurate pose predictions which resulted in
inaccurate depth.

The solution I settled on was to use the main camera for the pose network and
then only use the side cameras for the per camera depth models. This allowed for
a shared pose network across all camera angles and resulted in much better
depth.

### Highway Speeds

High speeds increase motion blur and increase the distance between frames which
make it harder for the model to learn since blur obstructs features (i.e. the
road surface becomes entirely uniform) and the larger distances makes close
objects move much further requiring highly precise depth/pose estimates to align correctly.

For highways especially they tend to be very uniform and the vehicles are
moving in the same direction as the road lines so there's not much difference
between frames.

{{% amp-img src="/monocular-improvements/highway-road.png" %}}
Road surface is inaccurate due to uniformity.
{{% /amp-img %}}

{{% amp-img src="/diy-self-driving/520_depth.png" %}}
This speed issue is also present at the edges of the main camera in the previous
post.
{{% /amp-img %}}

I was able to mitigate this issue by only training on captures below 50 mph.
This worked to avoid that issue but skipping training data is much less than
ideal.

### Image Space Depth Improvements

To maximize the amount of context for the images I tried feeding the full images
into the depth model and applying the loss on the full 1280x960 frames instead
of just the 640x480 frames. This helped a bit with the fine details but the
distance accuracies weren't significantly improved.

{{% amp-img src="/monocular-improvements/hrdepth-noz.png" %}}
Fine details such as the post are more accurate but speed corruption still
occurs.
{{% /amp-img %}}

For the higher resolution model I based my model off of HR-Depth which seems to
be the SoTA for monocular depth. https://arxiv.org/pdf/2012.07356.pdf

HR-Depth is better than monodepth2 especially for higher resolution images
but it's not hugely better.

### Masking

I switched to using a static mask instead of the automatic masking that was used
in monodepth2. A static mask seems to perform better for the sides of the
vehicle which are shiny and avoids accidentally masking out repeating patterns.

{{% amp-img src="/monocular-improvements/mask.png" %}}
Static mask for one of the repeater cameras.
{{% /amp-img %}}

Moving objects such as cars can be very problematic since standard Structure
From Motion training assumes all objects are static.

To try and handle moving objects such as cars I tried applying a semantic layer
to automatically mask out all moving objects to try and get the model to not
result in infinite/zero distances for cars. While this reduced the error it
still didn't return accurate results.

{{% amp-img src="/monocular-improvements/semantic-mask.png" %}}
Combination of semantic and static masks for a pillar camera.
{{% /amp-img %}}

### Radar Based Masking

To enable the model to learn some car distances I tried using the radar data as
a label but since the radar data is so noisy that didn't work out well at all.

{{% amp-img src="/monocular-improvements/radar2.png" %}}
Overhead traffic lights causing erroneous radar returns.
{{% /amp-img %}}

{{% amp-img src="/monocular-improvements/radar.png" %}}
The heights for the radar returns are often inaccurate which makes fusing with
image data hard.
{{% /amp-img %}}


A slightly better optimization was to use it as a filter on top of the predicted
bounding boxes to only train the model on stopped cars.

{{% amp-img src="/monocular-improvements/radar-mask.png" %}}
Top to bottom: The raw semantic layer bounding detections, the radar filtered
detections, and the resulting mask.
{{% /amp-img %}}

{{% amp-img src="/monocular-improvements/hrdepth-noz.png" %}}
Car depths are more reasonable but have some strange artifacting and aren't very
accruate.
{{% /amp-img %}}



### Multicamera Fusion

To actually use the depth maps from the side cameras I needed to merge them
together. I extended my existing 3D renderer to support rendering multiple
cameras as well as the depth maps.

{{% amp-img src="/monocular-improvements/multicam-single.jpeg" %}}
Projecting all of the cameras in one scene.
{{% /amp-img %}}

{{% amp-img src="/monocular-improvements/multicam-zoom.jpeg" %}}
Alignment is alright for mid range depths (~10-20m) when the car is driving
straight.
{{% /amp-img %}}

I switched the renderer to using meshes instead of point clouds and used the
semantic mask to mask out cars which add extra noise to the render.

{{% amp-img src="/monocular-improvements/fusion-two-lane.png" %}}
A two lane road. Alignment and accuracy is quite good for this.
{{% /amp-img %}}

{{% amp-img src="/monocular-improvements/fusion-forward.png" %}}
With some fine-tuning of the render the fused output is quite good. Edges of
objects aren't sharp resulting in jagged edges and noise.
{{% /amp-img %}}

{{% amp-img src="/monocular-improvements/fusion-alignment.png" %}}
As the points get further from the vehicle the accuracy gets much worse. The
edges of the images have less context so end up with inaccuracies.
{{% /amp-img %}}

These renders are a pretty reasonable but a bit cherrypicked since the car is
moving in a straight line, it's well lit and I'm significantly cropping the
edges of the images to avoid distortions near the edges. The results are much
worse when the car is turning or at night. Many of the rendering parameters
(i.e. frame spacing, cropping, distance thresholds) are situational as well.

The noise from edges of objects was a notable issue in my initial voxel
representation training set. Getting these renders to be robust requires good
localization and a lot of fine tuning to make sure they work in all cases.

{{% amp-img src="/monocular-improvements/colmap.jpeg" %}}
For reference here's the output from colmap. Generating this took two days on my 3090.
Fine details seem to be better than my result but uses stupid amount of
processing vs the monocular depth models which can run in real time.
{{% /amp-img %}}

### Multicamera Consistency

To improve the multi camera consistency in all situations I tried using the
cross camera projections from "Full Surround Monodepth from Multiple Cameras"
(https://arxiv.org/abs/2104.00152). This got my camera reprojection code honed
in quite accurately since I needed to be able to exactly map from one camera to
another.

This seemed like a promising approach since it would allow me to use monocular
depth models while still getting cross camera consistency.

{{% amp-img src="/monocular-improvements/projection-main-pillar.png" %}}
Projection from the main camera onto the left pillar camera.
{{% /amp-img %}}

These alignment and projections from one camera to another seem quite accurate
at first glance.

{{% amp-img src="/monocular-improvements/projection-repeater-backup.png" %}}
Projection from the left repeater camera onto the backup camera.
{{% /amp-img %}}

If you look at the difference between the projected image and the original
you'll notice that it never actually turns black which would indicate perfect
alignment. The exposure and vignette patterns of the cameras (i.e. brightness)
don't match up between the cameras unlike the dataset in the original paper.
This ended up with blurry depth outputs as it tried to make the brightness match
instead of the actual structure.

### Rectification

As part of the multicamera projections I needed to rectify the images.
Rectification eliminates the camera lens distortions and ensure that straight
lines in reality are straight. The fisheye/backup camera had the largest
distortions but there was lens distorions in all of the cameras.

This required having exact camera extrinsics (position, direction) as well as
camera intrinsics (fisheye distortion, focal lengths). I was able to calculate
the camera distortions by using OpenCV and a printed test pattern.

{{% amp-img src="/monocular-improvements/calibration-grid.png" %}}
Computing the distortion coefficients using a known test pattern attached to my
cutting board.
{{% /amp-img %}}


### Increasing Distance Between Frames

One solution to getting more accurate depths at further distances is to increase
the distance between the two frames when computing disparity. This would give
more movement on pixels that are far away which should result in more accurate
numbers.

Unfortunately as you increase distance between frames there's less common
features so it makes it harder for the model to learn nearby depth such as
road surfaces. It's the same problem as with higher speeds.

### Enforcing 3D constraints

A lot of the issues with the depth model is because purely image-space losses
don't enforce any real world constraints. This resulted in a lot of weird
behaviors such as the ground at high speeds, moving cars or reflective puddles
being at infinite distance. This just isn't physically possible.

Since I had already done all the work to figure out camera extrinsics and how to
map them into the real world for the multicamera reprojections I decided to try
and enforce a 3D constraint.

I converted each pixel to the XYZ coordinate and then grabbed the height. I
added a weak loss on the height to penalize points that were physically
impossible (i.e. meters underground). This gave the model a prior on what was
realistic vs not so for points on the ground that were previously infinitely far
away it forced the model out of the bad local minimum.

{{% amp-img src="/monocular-improvements/zloss.png" %}}
Main camera before vs after adding a zloss. Road surface is perfect.
{{% /amp-img %}}

{{% amp-img src="/monocular-improvements/zloss-wet.png" %}}
Wet road would normally result in incorrect depths due to the reflections.
{{% /amp-img %}}


### Conclusion

Using monocular depth models for full 3D reconstruction is very possible but it
requires a lot of work to get the models accurate enough to do multi-camera and
multi-frame fusion.

I got pretty good results but they weren't quite robust enough for me to feel
very comfortable labelling the output for lane lines etc. If you had some LIDAR
ground truth to do some fine tuning it would help a lot especially for the
further distances. For camera fusion, getting localization accurate is highly
important for these and I haven't spent enough time fine tuning it/don't have
enough time. Localization is very finicky and isn't as much fun as modeling.

Tesla has LIDAR data and engineers that can spend significant time fine-tuning
these approaches but for my use cases it's a big time sink to get them into a
robust and polished state.

For real world applications handling these in 3D via geometric constraints seems
to be the way to go and caused me to pivot to a new approach which is described
in the next post.
