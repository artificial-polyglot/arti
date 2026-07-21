# Arti2 Plans -- Architecture & General Design

Gary N Griswold, July 14, 2026

The primary purpose of Arti2 development is to add additional model training, inference, and error analysis intended to improve the accuracy of Arti.  While adding this functionality, it makes sense to apply lessons learned from the use of Arti to produce a more mature product.

## User Interface

The prescribed way to interact with Arti2 is to use its HTTP client.  This is an always-on service.  It includes an HTML form for submitting jobs that is similar, but more complete than the static form used today.  The HTTP client generates request.yaml files (or possibly request.json files) and submits them to a queue.

The user receives a notification of completion or failure of the job by email or SMS according to the means of communication selected on the HTTP client.  The email can be used to deliver some output.

All output is stored in an external service, such as AWS S3 or Cloudflare R2, and the HTTP client provides user access to that output.

The proofing and compare reports that are used for QA analysis of audios can be delivered by email, or accessed by the HTTP client.  These reports will not require the user to link the report to audio data on their local disk, the report will access the audio data in S3/R2 that was used for AI training.

The source code for the HTTP client will be stored in the same github repository as the arti server, because they share code that is used to check the correctness of a user's request file and input data.

## Component Architecture 

### Introduction

Arti is currently a single executable of many modules that are selectively executed by a controller according to the instructions from the user in the request.yaml file.  To make it easier for someone interested in just a part of Arti, new modules developed for Arti2 will be executables that can be run on their own.  This is also being done so that we can later add the ability to integrate externally developed components.

### Arti2 main

1) Arti2 main requires one command line parameter, the full text of the request.yaml (or json) config file.
2) Arti2 main will pass the bytes read to Arti2's ProcessV2(request []bytes)
3) Like now, the Arti2 request file must either indicate is_new: yes, in which case it creates a sqlite database, or it must explicitly include a database path in the request.
4) Arti2 will insert the request in json format into the sqlite database.
5) On completion, Arti2 ProcessV2 does not return content. (Docker makes that difficult)
6) Courier delivers output to R2/S3, and HTTP client delivers it to the user.

### Arti2 component

An Arti2 component is code with a main executable that is also callable by the controller.  It generally has one purpose, such as alignment by some algorithm, or training using some model, or speech to text by some method, or qa proofing by some logic.

1) A component requires one parameter, the local path to the request.yaml file.  A very experienced Arti user would be able to send the component only the subset needed to identify the inputs, outputs, and processing options that are important for this component.  But, it is simpler to just give each component an entire request file.

2) The component main(), passes the request local path to Component type, which does the following:
  a) It parses the request file.
  b) It copies in the remote database if needed
  c) And, it opens the remote database
  d) Or, it creates and empty database
  e) Inserts the request file into the requests table

3) The component main() passes the open database and the decoded request object into the component package code. For internal Arti2 components it passes the db.DBHandler, but future external components should get the *sql.DB object.

4) The component package code should return a list of paths for any output produced,
5) It also stores these in the database.

5) The component main() will print out this information.  And it will push these to a location on R2/S3

### Services

Arti2 contains basic services that are not part of any component.  All of these services already exist in Arti, the point being made here is that these will remain services that are part of the Arti2 core.

Services include:
1) Download text files, audio files, sqlite database, or model from S3/R2.
2) Reading .sfm, .usx, .xlsx, .xls files into scripts table
3) Parsing audio filenames, and inserting into audio_files table
4) Courier delivery of output to R2/S3.
5) Logging
6) Subprocess exec

There is a special case where someone might write the software to produce a sqlite database from sources Arti does not know how to process, and then use that database as input to an Arti2 run.

### Logging

To be more consistent with contemporary practices, the log will be changed to json format.  It will exist in two forms.  The log will be written to a local file so that the Courier can deliver it to R2/S3.  It will also be written to stderr so that it can be received by standard logging tools.

## Infrastructure

### Docker

Docker will be used to package the Arti2 server.  This will package Arti2 and all that it requires to process as a single image that can be loaded in a few seconds onto a serverless worker.  The HTTP client might also be packaged the same way in order to simplify deployment.

### R2/S3

In Arti2, the R2 or S3 services are the datahub where all input, output, and trained models are stored.  R2 appears to be much faster than S3, and charges no data transmission cost.  And Cloudflare is known for being a global provider of services at the edge.

### Serverless GPU

Runpod.io is the provider serverless GPU workers that is being used for development, but using Docker will simplify porting Arti2 to any provider, including back to AWS.

## Future

1) Arti currently runs only the modules requested by the user, but it runs them in a sequence that is hard-coded in the controller.  In the future, the request.json (or .yaml) file will become a procedural script, and components will be executed in the sequence specified by the user.

2) Arti currently can only run modules and components that are part of the arti repository.  In the future, a component registry will be added that will allow the use of externally developed components in Arti3.
