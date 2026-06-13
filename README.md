# Artificial Polyglot (arti)

An Application Server for checking correctness of Bible audio files, 
especially for low resource languages.

## Objective

Artificial Polyglot is an open source software service that validates audio Bibles 
by using AI to compare audio recordings against the original text.  
Developed for Faith Comes by Hearing, it has reduced the time required to proof 
an audio New Testament from 10-15 days to 2-3 hours.  The service, 
which is nicknamed Arti, was developed pro bono.  And has an MIT license attached.  
It is currently used to QA audio Bibles that are recorded in the field.

## Features

* Read Text - Supports USX, USFM (in test), Excel script, and CSV formats.  
All text absent from the audio recordings is automatically removed, 
such as footnote and cross references.
* Read Audio - Supports MP3, and WAV files, organized either as chapter files, 
or script line files.
* Verse Delineation - Aeneas (for comparison only), MMS Fairseq forced alignment, 
and MMS model forced alignment.
* MMS Adapter Model Training - Trains an MMS adapter model from the provided text 
and audio recording.  Samples that forced alignment detected that were likely 
errors are excluded from training.
* MMS Adapter Inference - Uses the trained adapter model to generate a 
transcription for an audio recording.
* MMS Model inference - Generates a transcription using the base MMS model 
without additional training, by searching a language graph to find the closest 
language that MMS supports.
* Difference Report - Produces a character-level difference report with an 
integrated audio player, enabling human reviewers to identify and confirm errors.
* Output Delivery - Results are stored, and delivered by email to the requesting user.

### Input Formats Supported

* USFM - Unified Standard Format, also called SFM
* USX - Unified Scripture XML
* XLSX - Audio Script Format
* XLS - Legacy audio Script Format
* CSV - Arti's Text Format

* WAV - Waveform Audio File
* MP3 - MPEG-1 Audio Layer III 

### Data Cleaning

While an XLSX script contains exactly the text recorded in the audio, 
the USFM and USX format contain additional text, such as introductory notes, 
footnotes, and cross references.  When Arti processes these two file types, 
it removes all of this text that will not be included in the audio so that 
the text will be identical to what is found in a script.  
It appears that other tools used require manual cleaning of USX or USFM 
prior to use.  Some testing has been done comparing Arti's cleaning with the 
manual cleaning being done, and early results indicate that Arti's cleaning 
is more accurate.

### Script Versification

Scripts are organized by script line, not verse in order that each script line 
contains the text of one speaker.  While the other text formats have data 
organized by verse.  Arti has the ability to read a script into verses as 
well as script lines, which is useful for tasks like doing verse delineation 
from a script.

### Audio Granularity

Audio files are recorded as script lines, and then later the script lines are 
combined into chapter files.  Arti is able to process audio script lines 
and audio chapter files.

## Audio Delineation

Arti provides a number of ways to do verse or script line delineation.  
Some of these are obsolete, but having these abilities provided a convenient 
way to check the accuracy of the various methods.

* Aeneas - A legacy method of computing timestamps.
* Fairseq - The forced alignment method used by the Meta MMS team when they 
developed the MMS model.
* MMS_FA - The multilingual forced alignment method built into the torchaudio 
library as a result of the MMS project.
* In Dev - Another method is in development that should be able to find 
timestamps for words in the audio that are not in the text.  
The two forced alignment methods are not able to do this.

### Character Level Timestamps

Arti uses an enhanced version of MMS_FA in order to capture timestamps for 
each character of text. While, verse delineation is often thought of as 
two numbers, a beginning and ending timestamp, this is an over simplification.  
The timestamps that best synchronizes an audio player to text, might not be 
the best timestamps to use when training a model.

Using character level timestamps, it is possible to deliver timestamps that 
are in any specific place, such as the midpoint of silence between two verses, 
or at any other point in the silence between words, such as the start of a word.

## Speech to Text

### Language Substitution

The MMS model knows 1300 languages, but most of these language's audio was 
produced years ago, before MMS was developed.  The languages to be processed today, 
are only occasionally one of the MMS supported languages.  
When a language to be processed is not supported by Arti, 
it searches a database of languages imported from glottolog.org 
to identify the closest language that is supported by MMS.  
It then does speech to text in that language.  This usually works 
surprisingly well.

### Decoders

The quality of speech to text is enhanced by using a choice of decoders.

* Greedy - Not actually a decoder, it just returns the highest probability 
result for each character.
* Simple - A simple decoder
* Hotwords - A decoder that has access to known words in the target language, and
attempt to return words in that language
* Kenlm - A post processing language model that has the entire script being processed,
and uses it to return a more accurate result

## Training

Arti trains an MMS adapter, which means that it uses a copy of the MMS model 
whose neurons are frozen and not altered by training, but an additional 
set of neurons are trained.  Hence it is called an adapter.

The MMS_FA forced alignment method produces a probability for each character 
that is is correct in the audio.  This data is used to identify verses that 
likely have errors, and these verses are removed from training, 
so that the model does not learn the error, but will identify the error 
when speech to text is done.

A couple other training methods are under investigation, and partially complete:
* Wav2Vec2CTC - verse samples - an investigation to see whether the presence 
of the MMS model is really a help when the model has been trained in a singe 
language.
* Wav2Vec2CTC - word samples - an investigation to see if training that 
excludes word order and other factors will be more accurate for learning 
the relation of sounds to text.

## Comparison

The original and official text is compared to the speech to text transcript 
using a Google developed product call diffpatchmatch.  This tool uses the 
Myers Diff Algorithm.

The standard output of this process is an HTML report that highlights differences,
and presents an audio play button so that the reviewer can make a 
final decision by listening to the audio.

This report is also available in json format for any use that needs to 
develop their own a different report.

## Processing

Because Arti accepts many different kinds of input and performs different 
processes on them, a config file is required input for each run that specifies 
the inputs, what tasks are to be done, and what outputs are to be produced, 
and who is to receive them.

While the config file approach works well when an automated system is making 
requests of Arti, for requests by human users, there is a web file interface 
where data to be processed is dragged into the form, and processes are 
selected by clicking radio buttons.

Each run of Arti produces a sqlite database that contains all of the interim 
values used in the calculation.  These are saved after each run, because it 
is an excellent resource for investigation problems or extracting additional data.

Because versification of a NT or speech to text take about 15 minutes each, 
and training takes 4-6 hours, an API design was not feasible.  Jobs submitted 
to Arti are placed in a queue; Arti does one job at a time, and delivers
requested output, and archives all output.  Using this queue feature, 
a batch of verse delineation jobs was recently run to update timestamps 
in Bible Brain for 2000 Bibles.

## Output

All output of the system is automatically cataloged into an S3 bucket.

## Architecture

Arti is a single Go executable in which a central controller orchestrates
a set of modules. For example, the text reading capability is implemented
as four separate modules: parse_usx, parse_usfm, read_script, and read_csv.
The AI components are written in Python. A user submits a job by providing
a YAML configuration file that specifies the input files, which modules to invoke,
and where to deliver the output. All modules read and write using a common SQLite
schema, making it straightforward to add new modules. Each run produces a fresh
SQLite database conforming to that schema.

## Architecture Plans

The current architecture has worked well and may continue to suffice.
However, if developers from multiple organizations need to contribute,
some architectural changes are necessary. A Request for Comments (RFC)
document has been written to propose such changes.

In summary, the RFC proposes a simple component architecture in which each
module is a standalone executable written in the developer's language of choice.
Each component reads a JSON input file or sqlite database from stdin and writes a 
JSON output file to stdout. The schema of these JSON files would adhere to 
standards agreed upon by the developers, so that modules developed independently 
can be immediately integrated.

Two schema approaches are under consideration. The first is a single master
schema that all modules share. The second is a family of schemas organized by
module type — one for reading text, another for model training, and so on.
A copy of the RFC is available on request. (gary at shortsands.com)




