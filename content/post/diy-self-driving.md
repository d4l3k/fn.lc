---
title: "DIY Self Driving - A Holiday Side Project"
date: 2022-01-10T20:23:09-08:00
---

_This work was done in collaboration with green, Sherman and Sid._

During the holidays I decided to take some time and try to make my own self driving machine learning models as part of a pipe dream to have an open source self driving car. I hacked this together over the course of about 2 weeks in between holiday activities.

>Disclaimer 1: I’m a software engineer on PyTorch but this work was done on my own time and not part of my normal duties.

> Disclaimer 2: The raw data was collected from a Tesla Model 3 but no Tesla NNs or outputs were used in this work. All training data and models were generated from scratch.


## Building From the Ground Up

So... how do you actually design a model so a car can understand the world? And even more importantly, how do you get training data for it?


### Inputs

To build all of this we have to start from the ground up. Here’s the raw data we can collect from the car:



* 9 Camera Feeds (1280x960 12bit RGB, 36fps)
    * 3 Front Facing (Main, Fisheye, Narrow)
    * 2 Pillar Left, Right
    * 2 Repeaters Left, Right
    * 1 Backup
    * 1 Selfie


{{% amp-img src="/diy-self-driving/5cam_residential.png" %}}
Example of 5 of the 9 cameras around the car (main, left/right pillar,
left/right repeater)
{{% /amp-img %}}


* Vehicle Info
    * Vehicle Speed (km/h)
    * Steering Wheel Angle (rads)
    * GPS Location and Heading (lat/lng, rads)
    * IMU Yaw, Pitch, Roll (rads/s)


{{% amp-img src="/diy-self-driving/speed.png" %}}
Vehicle speed information over time
{{% /amp-img %}}



### Collecting Data

I set up a daemon to periodically collect a short video and data capture every few minutes. This runs in the background in my car and generates training data over time. It’s not as diverse of a dataset as I would like but it’s good enough to train some interesting models on and I can always train with more data down the line.

This data was collected using a Tesla Model 3 which is a great platform for experiments like this but only if you can get root access which is extremely hard. I wish there was a Tesla approved way for everyone to get access to your own car like this.

The training data in compressed form is about 30GB of data. Unpacked and prepared for training as pngs/voxels it’s closer to 300GB.


## Understanding the World as an Image

This is a capture from the main camera. For training purposes I’ve scaled it to be 640x416 which allows a fair amount of detail but isn’t too large that I need a huge cluster of machines to train on.

{{% amp-img src="/diy-self-driving/main.png" %}}
A capture from the main camera in a residential area
{{% /amp-img %}}


This video feed has a lot of information about the world but to actually use it we need a training set. Classically you would have a large number of humans manually label the footage to pick out things like signs, cars, etc. Labeling video footage is super time consuming and I don’t have thousands of data labellers to help me out.

These images are in 2D but for driving tasks we really need to understand the world in 3D so labeling single images isn’t going to cut it.

To get a 3D representation of the world a lot of companies are using LIDAR on their vehicles. I don’t have a LIDAR unit so we’re stuck with just cameras and need to figure out a way to get a 3D understanding of the world from it.

One way to understand the world is to generate a depth map from the camera footage.



<p id="gdcalert4" ><span style="color: red; font-weight: bold">>>>>>  gd2md-html alert: inline image link here (to images/image4.png). Store image on your image server and adjust path/filename/extension if necessary. </span><br>(<a href="#">Back to top</a>)(<a href="#gdcalert5">Next alert</a>)<br><span style="color: red; font-weight: bold">>>>>> </span></p>


![alt_text](images/image4.png "image_tooltip")

{{% amp-img src="/diy-self-driving/residential_depth.png" %}}
Main camera and the predicted depth map from the monocular depth model
{{% /amp-img %}}


To train a model we typically need a dataset with input/output pairs to have the model match. In this case I _don’t_ have any labeled data much less high resolution depth maps from LIDAR. That leaves self-supervised methods to use.

Self-supervised methods use some inherent property to ensure consistency between the data. There’s a couple of different strategies here but here’s the two most effective ones from what I’ve seen:



* Stereoscopic imagery — this takes two known cameras from slightly different positions and calculates the “disparity” which can be directly converted to distance.
* Depth from motion — this takes sets of frames from a moving camera and using projections ensures that the depths from each frame is consistent.


### Stereoscopic Imagery

I’ve shown one of Tesla’s monocular depth models before which I now believe was trained using stereoscopic images between the main and fisheye cameras. [https://twitter.com/rice_fry/status/1415034007222317057](https://twitter.com/rice_fry/status/1415034007222317057)


{{% amp-img src="/diy-self-driving/fisheye_main_heater.png" %}}
Main and fisheye cameras which could be used to predict depth using stereoscopic
methods
{{% /amp-img %}}


Using the fisheye camera would explain why there’s an odd distortion in Tesla’s models where the camera heater wire is. It’s very faint on newer cars but still notable enough to affect stereoscopic models.

Getting these two lined up so you could do stereoscopic training is something I haven’t done before so I didn’t dig too much into this. You’d need to crop it to have the same field of view and then possibly rectify to counteract the fisheye distortion.

The cropped fisheye also has less detail than the full frame so you might not be able to get as high resolution of a model.


### Depth From Motion

I picked training the model using depth from motion for these purposes since it allowed me to train models using just a single camera which is important since we only have multiple overlapping cameras facing the forward direction.

I ended up using monodepth2 ([https://github.com/nianticlabs/monodepth2](https://github.com/nianticlabs/monodepth2)) as a base since it’s quite effective out of the box. Unfortunately, most depth from motion models assume that everything is static so if you have a moving car in the shot the predicted depth will be much too close or much too far depending on whether it’s moving towards or away from the camera. Stereoscopic setups avoid this since they compare two cameras at the same time.


{{% amp-img src="/diy-self-driving/520_depth.png" %}}
Main camera and predicted depth with a moving vehicle with incorrect depth
{{% /amp-img %}}


This shows the moving car at the wrong depth and has some issues from the edges of the camera lens. To output static terrain it’s straightforward to apply a multiple frame post-processing to ignore both of these cases.


### Calibrating Model Depth to Real World Distances

The other issue with depth from motion is that it’s not tied to the real world. The predicted distances are all relative to each other and not tied to any one specific model.

{{% amp-img src="/diy-self-driving/monodepth2_networks.png" %}}
Depth and pose networks. Credit: monodepth2 paper [https://arxiv.org/abs/1806.01260](https://arxiv.org/abs/1806.01260)
{{% /amp-img %}}


Fortunately I was able to come up with a simple fix for this. Monodepth2 uses two networks: the first generates the depth from the frame; the second calculates how the camera moves through space. Using the pose camera it can project two paired frames and ensure that the difference is equal to the motion through space.

These motions aren’t tied to actual distances so aren’t really usable in the real world. However, in this case we know how fast the car is traveling so we know the actual distance between the two frames. We can take the output from the pose network and the real distance the car traveled to scale the depth output to be in real world meters.


## Understanding the World as a Point Cloud

We have a depth model that has pretty reasonable output and is tied to real world distances. Where to now?

We know the motion of the car (speed, rotations) and we can compute depth from a still image. We can combine the two with the original video and generate full point clouds of our video clips.

NOTE: These projections are _just_ from the main camera. Using all of the cameras would greatly improve the detail on the edges (side cameras) as well as improve the range (narrow) allowing for high res reproductions of complex intersections.


{{% amp-img src="/diy-self-driving/520_pointcloud.png" %}}
Point cloud from WA-520 — the moving car from the picture above is completely gone and the signs are readable. See more: [https://streamable.com/f5prxc](https://streamable.com/f5prxc)
{{% /amp-img %}}



{{% amp-img src="/diy-self-driving/residential_pointcloud.png" %}}
A residential scene with many objects. See more: [https://streamable.com/x2vce2](https://streamable.com/x2vce2)
{{% /amp-img %}}

These frames aren’t quite perfectly aligned since my localization is fairly rough but things are generally where they should be and the road lines line up across multiple frames. It’s pretty amazing that you can render 24 million points in a web browser.

I’m only rendering every 4th frame to not destroy my laptop. A full render with all cameras and every frame would have tons of detail.


## Understanding the World through Birds Eye View

If you take these projections and just look at the ground it’s quite feasible to quickly label and generate birds eye view maps such as used by Tesla in their FSD Beta software.

Here’s that freeway scene from above:



{{% amp-img src="/diy-self-driving/unlabelled_bev.png" %}}
The birdseye view of the generated point cloud showing the road surface
{{% /amp-img %}}


And here’s it quickly annotated using GIMP:


{{% amp-img src="/diy-self-driving/labelled_bev.png" %}}
The birdseye view with manual annotations
{{% /amp-img %}}


Labeling video clips in 3D like this makes it much easier to label. Drawing lane lines and road edges across the whole clip with proper tools would only take seconds compared to manually annotating lane lines on hundreds or thousands of individual frames.

Sadly there’s no good open source tool I’m aware of for labeling birdseye view frames like this.

Unlike the full point clouds, these birdseye maps are computationally cheap enough to run in realtime in a car which makes them a great middle ground.


## Understanding the World as Voxels

I didn’t really want to spend my vacation labeling data so instead of creating a BEV network to predict lane lines I decided to try and generate full voxel representations of the world instead.


> This is inspired by how Tesla’s general static obstacle networks work that I showed at [https://twitter.com/rice_fry/status/1463628670208147460](https://twitter.com/rice_fry/status/1463628670208147460)


{{% amp-img src="/diy-self-driving/voxel.png" %}}
Rasterized voxel representation of the generated point clouds
{{% /amp-img %}}


I can take the point clouds and rasterize them into a dense 3D grid of voxels around the vehicle at each time position. For this model I generated a grid of (144, 80, 12) which equates to (48m, 26.6m, 4m). This is on the smaller size but I wanted to start small to test out and the point clouds are only generated from the main camera so the range/width isn’t that great for now.



{{% amp-img src="/diy-self-driving/5cam_residential.png" %}}
The five cameras used as inputs to the voxel network
{{% /amp-img %}}


As the input I feed in five cameras facing around the car. This is the main, the front left and right pillar cams and the rear facing left and right repeater cameras. This gives an almost complete 360 view around the car.

For the output, I just feed in the generated voxels from point clouds.


{{% amp-img src="/diy-self-driving/voxel_preds.png" %}}
Example: Ground Truth (left); Predicted from cameras (right)
{{% /amp-img %}}


This clip is fairly hilly so the curvature of the road shows up in the voxel data. I had to model the pitch/yaw of the car in my projections as well as heading to get the data to align across multiple frames.

Here’s a video of the voxel output: [https://youtu.be/bwos0XsTGUg](https://youtu.be/bwos0XsTGUg)

This model is probably overfitting to the data since I don’t have the largest dataset. After post processing I have ~15.2k frames/voxel training examples which is only about 7 minutes of footage though it’s from a bunch of different roads.

This is the network I came up with:

{{% amp-img src="/diy-self-driving/VoxelNet.svg" width=1258 height=675 %}}
My DIY BEV Voxel Net Architecture
{{% /amp-img %}}


It’s loosely based off of the Tesla AI day presentation with a lot of guesswork. It seems to work reasonably well and can be trained with or without the depth encoders frozen. I use the trained depth encoder that generated the monocular depth maps.

Due to the size of the feature maps I’m only able to pass in the f3 and f4 feature levels to the transformers. With this network I can achieve 97.5% train accuracy. I don’t currently have a proper val/test accuracy since I was primarily focused on getting something that can converge.

The model is trained using per voxel binary cross entropy loss which seems to work well enough. A number of papers train their BEV models with a GAN style discriminator but that didn’t seem to be necessary here.

There is a lot of noise in the training data that is probably stopping the model from learning all the fine detail but the high level structures are definitely learned. There’s some more work that really needs to be done on the training data generation to better weigh depths closer to the camera more than those far away.

Here’s the full model pytorch model definition: [https://gist.github.com/d664de8f68607992edc7e09c1991d131](https://gist.github.com/d664de8f68607992edc7e09c1991d131)


## Learnings

* PIL (the go to PyTorch image loader) doesn’t support 16 bit images and silently down samples to 8 bit which is problematic for training on HDR/RAW video. Need to use something like opencv to load which preserves the full 16 bit values
* Transformer/MultiheadedAttention interfaces are really designed for language modeling. It’s pretty clunky to use them with CV models and require a lot of reshape/permute to get the dimensions to line up correctly.
* There's not any good patterns/solutions for 2D -> 3D transforms. I.e. converting depth images to point clouds to voxel representations (lots of custom Go code for performance)
* No good open source 3D data labeller. Many datasets from academia only have 2D annotations and aren’t labeled in 3D (BEV/bounding boxes) so a lot of focus is generating 3D from 2d image space which is inherently less accurate than operating in 3D from the beginning.
* Single machine training can produce good results even for fairly large datasets and models. I did give in halfway through and buy myself an RTX 3090 since my 8gb 3070 Ti only had enough memory for a batch size of 1-2.
* It’s much cheaper to increase the transformer embedding dimension compared to increasing the number of K/V pairs.

## Next Steps

I’m planning on polishing this code up and sharing it on GitHub. I’d love to create an open source self driving stack though that's an incredible amount of work to do end to end.

I’m also planning on doing more improvement to clean up the point clouds more so I can improve the model accuracy. The depth model I’m using can’t handle dynamic objects so something like [https://arxiv.org/abs/2110.08192](https://arxiv.org/abs/2110.08192) or [https://arxiv.org/abs/2107.07596](https://arxiv.org/abs/2107.07596) may work better for this situation.

A small 3d labelling tool would also be really useful for extending this work to
support other birdseye view models such as road lines.

Thanks for reading this far! Happy to answer any questions or comments :)
