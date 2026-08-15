(function () {
	var activeButton = null;
	var activeAudio = null;
	var endTimestamp = null;

	function resolveSource(audioFile) {
		var dirInput = document.getElementById('directory');
		if (dirInput && dirInput.value && !/^https?:\/\//i.test(audioFile)) {
			return dirInput.value.replace(/\/+$/, '') + '/' + audioFile;
		}
		return audioFile;
	}

	function stopActive() {
		if (activeAudio) {
			activeAudio.pause();
			activeAudio.ontimeupdate = null;
			activeAudio.onloadedmetadata = null;
			activeAudio.onended = null;
		}
		if (activeButton) {
			activeButton.textContent = 'Play';
		}
		if (window.avOnVerseStop) window.avOnVerseStop();
		activeButton = null;
		activeAudio = null;
		endTimestamp = null;
	}

	function startVerse(button, audioFile, beginTS, endTS) {
		var audio = document.getElementById('validateAudio');
		activeButton = button;
		activeAudio = audio;
		endTimestamp = endTS;
		button.textContent = 'Pause';

		audio.pause();
		audio.playbackRate = (typeof window.avPlaybackRate !== 'undefined') ? window.avPlaybackRate : 1;
		audio.src = resolveSource(audioFile);

		audio.onloadedmetadata = function () {
			audio.currentTime = beginTS;
			audio.play().catch(function (err) { console.error('play failed', err); });
		};
		audio.ontimeupdate = function () {
			if (window.avOnTimeUpdate) window.avOnTimeUpdate(audio, audio.currentTime);
			if (endTimestamp !== null && audio.currentTime >= endTimestamp) {
				stopActive();
			}
		};
		audio.onended = function () {
			stopActive();
		};

		if (window.avOnVerseStart) window.avOnVerseStart(button, beginTS, endTS);
	}

	window.playVerse = function (button, audioFile, beginTS, endTS) {
		if (activeButton === button && activeAudio) {
			if (activeAudio.paused) {
				activeAudio.play().catch(function (err) { console.error('play failed', err); });
				button.textContent = 'Pause';
			} else {
				activeAudio.pause();
				button.textContent = 'Play';
			}
			return;
		}
		stopActive();
		startVerse(button, audioFile, beginTS, endTS);
	};
})();
