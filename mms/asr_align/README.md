
The asr_align is an unfinished experiment to do speech to text without first breaking it into verses.

This method could be applied to all speech to text modules, and would cut the time to perform that task in half.

The current code is capable of doing a transcription of a chapter.

TODO
1. XXReview asr_align.go and asr_align.py to how they are similar or different from mms_asr. 

Q. transcription can be stored in scripts table.  Should it be?

What needs to be done for further testing.
1. Create new test method
2. In Test Create database in :memory:
3. Alter table add transcription, diff text 
4. read text of MZJ 
5. Select Text as string, include {n} verse markers 
6. Diff the script text to transcript 
7. Break result into verses (should lines be an option)
8. Store the verse aligned transcription and diff text into database
9. Generate Pairs slice 
10. Generate report from pairs using existing code

* Also, Need to run ASR2 over entire NT and OT to ensure it is able to process all chapters.  
It is likely that large chapters will need to be split into chunks (with overlap),
and reassemble stitching the transcription somehow.

Notes:
5/4/26 asr_align.py is identical to mms_asr.py except that asr_align.py read the audio using Dataset
which Claude considers slow, becuase it is instantiated on each sample.
5/4/26 Add debug statement to asr_align.py to check size of numbers returned.