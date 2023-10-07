---
title: "torchdrive: Open Source + Nuscenes Support"
date: 2023-10-06T21:01:23-07:00
---

Over the past 9 months, I've been rewriting my models from the ground up and
open sourcing them on GitHub. The code is now [fully public and available for
anyone to use and modify](https://github.com/d4l3k/torchdrive/tree/main).

[![torchdrive](/torchdrive/torchdrive.svg)](https://github.com/d4l3k/torchdrive)

The majority of the code is a
[BSD-3-Clause license](https://github.com/d4l3k/torchdrive/blob/main/LICENSE)
which matches other open source projects such as PyTorch. There are a few pieces
with modules borrowed from other projects.

_This is a follow up to
[3D Dynamic Objects]({{< ref "3d-detr.md" >}})
and is part of a series where I try to train models to perform common self
driving tasks from scratch._

## NuScenes

Thanks [David E.](https://github.com/dav-ell) to we now have full
[NuScenes dataset](https://github.com/d4l3k/torchdrive/blob/main/torchdrive/datasets/nuscenes_dataset.py#L320)
integration in torchdrive so anyone can download the datasets and train the same
models.

NuScenes provides video from 6 cameras around the vehicle as well as Radar and
LIDAR data. torchdrive only needs the raw video footage so we only consume the
camera data and vehicle position as inputs and don't use the LIDAR or any
labeled data.

{{% amp-img src="/torchdrive/nuscenes-example.png" %}}
Example views from CAM_FRONT_LEFT, CAM_FRONT, CAM_FRONT_RIGHT and the
corresponding outputs from a multi task torchdrive model.
{{% /amp-img %}}

You can learn more about Nuscenes from the
[official documentation](https://www.nuscenes.org/nuscenes)
as well as in the
[torchdrive dataset](https://github.com/d4l3k/torchdrive/blob/main/torchdrive/datasets/nuscenes_dataset.py#L320).

## Multitask Architecture

torchdrive has been rewritten to provide cleaner interfaces between the
different components. There's three primary concepts:

### Camera Encoder

The camera encoder takes in the raw image data and produces an embedding
with dimensions `[ch, h, w]`. Any encoder can be used though torchdrive uses a
RegNet with a partial FPN by default.

### Backbone

The backbone takes in the camera features as well as the camera extrinsics and
extrinsics and produces two outputs.

Voxel: A full voxel grid with an embedding at each voxel with an arbitrary
number of features per voxel. By default, we use a grid of `[24, 256, 256, 16]` with
scale of 1/3 meters with 24 channels.

Embedding: A high channel low grid size embedding that can be used for sparse
tasks such as 3D object detection or trajectory prediction. By default this is `[256, 16, 16]`.

There's currently 3 backbones:

1. rice -- a custom transformer based backbone that uses transformers to
   learn the camera to BEV transform.
2. simplebev -- a SimpleBEV based implementation that uses camera
   projections to create a birdseye view embedding.
3. simplebev3d -- I'll be writing a blog post about this in more detail.

The models are at: https://github.com/d4l3k/torchdrive/tree/main/torchdrive/models

### Tasks

torchdrive is designed to handle arbitrary tasks that consume 3D data. I've
completely rewritten this interface to be able to consume the backbone outputs
and handle multiple subtasks.

Out of the box, there's 4 subtasks:

1. [voxel](https://github.com/d4l3k/torchdrive/blob/main/torchdrive/tasks/voxel.py)
   -- consumes the voxel output and predicts per voxel occupancy, semantic
   classes and velocity by using SFM + semantic losses. This is by far the most
   complex task at 1k+ lines.
2. [det](https://github.com/d4l3k/torchdrive/blob/main/torchdrive/tasks/det.py)
   -- consumes the high level embedding and predicts 3d bounding boxes by
   projecting 3D DETR bounding boxes into image space.
3. [path](https://github.com/d4l3k/torchdrive/blob/main/torchdrive/tasks/path.py)
   -- a simple transformer based trajectory prediction model.
4. [ae](https://github.com/d4l3k/torchdrive/blob/main/torchdrive/tasks/ae.py)
   -- uses inverse of the rice backbone to predict the original image (this
   doesn't work very well)
5. (not landed) mesh model + drivable space and lane lines. This hasn't been
   ported over yet but if folks have interest, please file a GitHub issue and
   I'll publish the code for it.

The tasks are all at: https://github.com/d4l3k/torchdrive/tree/main/torchdrive/tasks

## Config Management

torchdrive now uses Python config files located in the
[configs](https://github.com/d4l3k/torchdrive/tree/main/configs) directory.
These specify the dataset, encoders, backbone and the tasks to train.

These configs are the configs that I use to train my models so should allow
anyone to reproduce my models rather than requiring eldrich command line
invocations (sorry!). These will evolve over time as the models improve and
should be relied upon to be stable.

These are fully unit-tested and can be run via `pytest torchdrive/test_train_config.py`.

## Installation

Please check the Dockerfile.cpu and requirements.txt for details on which
libraries and versions need to be installed. This process is complex due to
dependencies on mmlab and [bdd100k-models](https://github.com/SysCV/bdd100k-models).


## Distributed Training

To improve training performance for these models I've picked up a second GPU and
thus torchdrive now has distributed training support by integrating in with
[torchx](https://github.com/d4l3k/torchdrive/blob/main/.torchxconfig) and
TorchElastic.

Running a dual GPU training job locally with the standard configuration is now
as easy as running:
```
torchx run -- -j 1x2 -- --output experiments/my_experiment --config simplebev3d
```


## Contributing

### Running Tests

Tests can be run via:

```
pytest
```

These tests don't require GPU and can be run fully using CPU.

### Lint / Pyre

Lint and type checking can be run via:

```
scripts/lint.sh
scripts/docker_pyre.sh
```

