
The asr_align is an unfinished experiment to do speech to text without first breaking it into verses.

This method could be applied to all speech to text modules, and would cut the time to perform that task in half.

The current code is capable of doing a transcription of a chapter.

TODO
1. XXReview asr_align.go and asr_align.py to how they are similar or different from mms_asr. 

Q. transcription can be stored in scripts table.  Should it be?

What needs to be done for further testing.
1. There might be a problem that when chars are in the audio, but not the text, it is difficult to figure where they go.
2. If they are part of a work, they should stay with the word.
3. If they are a word themselves, then I am not sure what to do.

* Also, Need to run ASR2 over entire NT and OT to ensure it is able to process all chapters.  
It is likely that large chapters will need to be split into chunks (with overlap),
and reassemble stitching the transcription somehow.

Notes:
5/4/26 asr_align.py is identical to mms_asr.py except that asr_align.py read the audio using Dataset
which Claude considers slow, becuase it is instantiated on each sample.
5/4/26 Add debug statement to asr_align.py to check size of numbers returned.