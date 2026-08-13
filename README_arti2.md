# Arti2

The development of an updated Arti service started early May 2026.  Possibly it will be called Arti2.

### Motivation

The objective is to try and improve the accuracy of audio Bibles by identifying more errors.  My motivation is simply 2 Tim 3:16 and Rev 22:19.  When AI begins to produce audio Bibles by text-to-speech, as much QA accuracy as possible will be needed.

### Features

In May, I successfully computed the probability for each character and word that it was correct in the audio.  This made possible a new kind of report that presents the correct text with word highlighting to reflect the probability they are incorrect in the audio.  In preliminary testing this report finds all of the same errors as the past reports, but contains about 1/2 as many false positives.  I am hopeful it can be used to find additional errors.

In October 2025, I experimented with other means of training, which seemed to improve the accuracy of speech-to-text.  The problem with the current method is that the model is not constrained to learn only the relation of characters to sound, but learns other things like word order as well.  This additional learning is degrading the model's ability to QA audio files accurately.  There is more to be done to get this experiment ready for production use.

### Architecture

Each new feature is being developed as a component that can be run both as a stand-alone executable and as a module within Arti.  This is done to benefit those who will learn from Arti2 to reimplement similar components within their own systems.  I am delighted in whatever way these methods are used to serve our Lord's purposes.

### Infrastructure

Cloudflare R2 is being used for data storage rather than AWS S3, because Cloudflare is leading the industry in Internet security, and they are one of the highest performing cloud services, especially in remote locations, and are lower cost than AWS S3, by a significant margin.  The code I have written to read and write to Cloudflare R2 will run on AWS S3 with only a configuration change.

Docker is being used to build the Arti image so that it can be run server-less paying for only the minutes used while it is processing. It takes cloud services just a few seconds to provision a computer with the Arti Docker image, and they deprovision it in seconds if there is no other Arti job in the queue.  As a result, the cost of a full Arti run of a) verse delineation, b) 16 epochs of training, c) speech to text, d) comparison is costing about $12.  These Docker images plus environment variables can be run on any cloud provider with very little preparation.  Using Docker means operations groups, like HiQ, will not do complex provisioning.  Also, Docker makes it very simple to deploy any number of concurrent servers.

### Web

While Arti runs only while a job is being processed, it is best to have "always on" services available over the web.  A web service to deliver results, view models, input, and output, including viewing, playing and downloading those objects has been developed.  I intend to rewrite Jon Stearley's HTML based job submission web page to become a web service.  A third web service is planned to monitor the Arti queue, and jobs while they are processing.  On Cloudflare these web services will most likely cost nothing except the cost of R2 data storage.  They could be rewritten for AWS, but AWS will charge for using those features.

### Dependencies

Help from Gordon Fiddes's group at Faith Comes by Hearing will be essential to this project in order to verify the accuracy of the planned improvements.

### Applications

When the current objectives are completed, hopefully Gordon's group at Faith Comes by Hearing will find the new features useful, and wish to have it installed on a staging server.

I recently became reaquainted with a childhood friend, who is now a trained Linguist and expert in Thai and Laotian languages.  His work should benefit from using Arti.

Now that Arti development is in the "cloud", it would be practical to provide Arti services to anyone who wishes.  The means of controlling access to the buckets for FCBH test data can be duplicated for each user.

When an OBT translator is recording a language for which there are closely related languages or dialects in written form, they could be using Arti to see a comparison of the OBT audio to the text of the related language.  To facilitate, a number of optional features were recently added to Arti2 speech to text to increase accuracy when the text is more complete than the audio.

Using Arti at recording locations is the end game that will make the most difference in accuracy.  The simplest solutions involve having Arti read and write data from those system's datastores.  It will only be possible for me to help FCBH or anyone else with this when I have their cooperation in that development.

### Source code

The Arti2 development code is stored at https://github.com/artificial-polyglot/arti
Anyone wishing to identify specific functions, should contact me with questions, or consult https://deepwiki.com/artificial-polyglot/arti, or simply ask a good quality AI coding agent to study it and give you a report.

### Goals

It is my hope the Lord will preserve by ability to do this work and complete Arti2 by my 80th birthday in Feb 2027.