---
date: 2017-07-08T00:34:19-07:00
title: nwHacks Machine Learning
image: /images/nwhacks-ml/training.png
tags:
  - ubc
  - nwhacks
  - go
  - machine learning
  - hector
  - random forests
  - gbdt
---

I've been doing a bunch of work during my internship with Machine Learning
models so I figured I take a crack at applying them to some of my personal
projects. Just for fun I wanted to see what would happen if I tried to train a
model on the registration, check-in and submission data for nwHacks.

I decided to use [Hector](https://github.com/xlvector/hector), a suite of
algorithms completely written in Go since that's what most of the nwHacks
tooling is written in. Took a few hours to write a pipeline that would read in
the nwHacks registration data from Firebase and output it into a format that
Hector supports.

{{% amp-img src="/images/nwhacks-ml/training.png" %}}
Training the model and evaluating my personal results.
{{% /amp-img %}}


## Features

These are all the features that the model looks at. GitHub information is
queried from the GitHub API. Distances are calculated by geocoding the city
using the MapQuest API and calculating great circle distance from Vancouver.

* First hackathon
* Wants to mentor
* Listed GitHub
* Listed LinkedIn
* Listed personal site
* Number of teammates
* Name of school
* Distance from vancouver in KM
* Number of GitHub repos
* Number of GitHub gists
* Number of GitHub followers
* Number of GitHub users followed
* Bag of words using frequency
  * Resume text
  * Reason for going
  * GitHub repo descriptions

To check whether or not someone submitted, I exported the submission data from
the [devpost](https://nwhacks2017.devpost.com/) and did string matching on names
and email addresses from people listed. This is probably missing a bunch of
people who worked on the project but weren't listed on the submission.

## Algorithm

To figure out which algorithm to use, I ran 10 way cross validation with some
common machine learning algorithms.

### Receiver Operating Characteristic Curves

The models are graded using
[receiver operating characteristic](https://en.wikipedia.org/wiki/Receiver_operating_characteristic)
curves by measuring the total area under the curve (ROC AUC). This is a rough
approximation of how accurate the model is.

[One rule of thumb for AUC](http://gim.unmc.edu/dxtests/roc3.htm) is:

* .90-1 = excellent (A)
* .80-.90 = good (B)
* .70-.80 = fair (C)
* .60-.70 = poor (D)
* .50-.60 = fail (F)

An AUC of 0.5 means the model is completely worthless and is pretty much just
randomly guessing.

### Algorithm Comparisons

After inspecting some initial results I ended up settling on Random Forests as
they had the best performance with more complex features.

#### Without GitHub Data

| Model | ROC AUC |
|---|---|
| svm | 0.575548324252739 |
| rf | 0.7826494507380344 |
| knn | 0.6437386773574082 |
| gbdt | 0.7949250288350634 |

After these results I only looked at random forests and gradient boosted
decision trees.

#### With GitHub Data

| Model | ROC AUC |
|---|---|
| rf | 0.7781795846146551 |
| gbdt | 0.7476873539057933 |

## Results

The model for check-in and submission has very poor accuracy where as the model
for acceptance is reasonably accurate. Intuitively this makes sense since humans
are deciding whether or not someone gets accepted, but we're pretty bad at
estimating whether or not someone is actually going to submit a project or show
up. Those models are a bit better than randomly guessing but not by much.
There's also about 1/4th the amount of data for the check-in and submission
models.

| Classifier | ROC AUC | Personal Probability |
|---|---|---|
| Probability of check-in given accepted | 0.5215234069947506 | 0.618993 |
| Probability of submitting a project given checked-in | 0.5509568462997873 | 0.395492 |
| Probability of being accepted | 0.8470575003026127 | 0.862843 |

### Personal Performance

To evaluate the personal probability I've listed the input features below so you
can judge me yourself.

```go
reg := &db.Registration{
  Name:           "Tristan Rice",
  School:         "University of British Columbia",
  City:           "Vancouver",
  GitHub:         "d4l3k",
  LinkedIn:       "d4l3k",
  Reason:         "I really want to come to nwHacks and make some cool
                   stuff! I've gone that past couple of years and really
                   enjoyed it.",
  Resume:         "https://fn.lc/resume.pdf",
  Mentor:         true,
  FirstHackathon: false,
  Teammates:      "jinny, roy",
  PersonalSite:   "https://fn.lc",
  Email:          "rice@fn.lc",
}
```

## Other Random Thoughts

### Live Score on Registration Form

It might be interesting to list the probability of acceptance on the
registration form with a "hint: add more data to increase probability of
acceptance." However, that's probably a terrible idea.

### Sorting and Filtering Registrations

Could also be helpful for reviewers to use the models to quickly weed out any
very low effort submissions. It would definitely lower turn around time from
registration close to sending out acceptances. I'd have to try and improve the
model more and maybe remove the school and distance to make it more fair for
more distant applicants. There's probably a skew towards local students since
remote ones tend to not show up as often without travel reimbursement.

### Parameter Sweep

These models were all trained using Hector's default parameters. If we were
going to use these models it would definitely be worth it to run some large
scale parameter sweeps using bayesian optimization to tune the hyper parameters.

### Older Data

Might also be worth it to dig up the data from the 2015-2016 nwHacks which would
roughly double our training examples.

## Source Code

The source code is available at:
https://gist.github.com/d4l3k/d76c1f63027bd404d3e7357c7d575cbd under the MIT
License.
