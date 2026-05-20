# Filename Standards of Arti
### Last documented May 20, 2026

## USFM

2 numeric char sequence number + 3 char book code + other data + (.SFM or .sfm)

## USX

3 numeric char sequence number + 3 char book code + (.usx or .USX)

or

3 char book code + (.usx or .USX)

## XLSX

The filename is not parsed, by Arti + (.xlsx or .XLSX)

## Audio for USX or USFM

These are not parsed by char position, but split at an underscore char

{drama}_{ios}_{version}_{seq}_{book code}_{chapter}.mp3

drama is 01 for single speaker no music, 02 for multi speaker and music

iso is the language code

seq a sequence number to keep files sorted correctly, i.e. GEN chapter 3 is 003

## Audio for XLSX

These are also split by underscore, and the second to the last position is 5 char long or longer.  
This field is the script line number.

All of the other fields are filled in by data read in the script file.

## Bible Brain Version 4

The files are parse by splitting on underscore

{mediaid}_{A/B}{ordering}_{book code}_{chapter start}[_{verse start}-{chapter stop}_{verse stop}].mp3|webm

The square brackets indicate optional parts of the filename

e.g.
ENGESVN2DA_B001_MAT_001.mp3  (full chapter)
IRUNLCP1DA_B013_1TH_001_001-001_010.mp3  (partial chapter, verses 1-10)

## Bible Brain Version 2

These are parsed by precise character position

0 - A is OT, B is NT
1-3 is file sequence number

5-7 is chapter number

9-20 is book name (Note: book name, not book code)

21-end is Bible Brain MediaId





