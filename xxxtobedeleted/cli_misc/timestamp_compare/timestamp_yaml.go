package main

const BBTS = `is_new: yes
dataset_name: BBTS_{bibleId}
bible_id: {bibleId}
username: TS_DEV
email: gary@shortsands.com
output_file: BBTS_{bibleId}.sqlite
text_data:
  bible_brain:
    text_plain_edit: yes
audio_data:
  bible_brain:
    mp3_64: yes
timestamps: 
  bible_brain: yes
testament:
  nt_books: [{books}]
`

const Aeneas1 = `is_new: yes
dataset_name: Aeneas1_{bibleId}
bible_id: {bibleId}
username: TS_DEV
email: gary@shortsands.com
output_file: Aeneas_{bibleId}.sqlite
text_data:
  bible_brain:
    text_plain_edit: yes
audio_data:
  bible_brain:
    mp3_64: yes
timestamps:
  aeneas: yes
testament:
  nt_books: [{books}]
`
