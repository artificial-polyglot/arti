# Decides after each epoch whether training should continue, using the validation CER.
# NumEpochs from the request is the most epochs that will run, and the linear learning rate
# schedule decays over all of them.  These constants may later become request parameters.

import math

STOP_METRIC = "all"         # validation group whose CER decides: "fa_score", "silence" or "all"
PATIENCE = 4                # stop after this many epochs in a row without a real gain
MIN_RELATIVE_GAIN = 0.005   # a gain is real when CER falls by at least 0.5% of the reference CER
MIN_EPOCH_FRACTION = 0.75   # no stop before this fraction of maxEpochs, so the linear LR has
                            # decayed well down first; patience is still counted before it
MAX_EMPTY_FRACTION = 0.5    # while at least this fraction of transcripts is empty, the model has
                            # not yet left the all-blank phase, and patience is not counted
BLANK_FAIL_UPDATES = 4000   # training fails if it is still in the all-blank phase after this many updates


class EarlyStopper:
    def __init__(self, maxEpochs):
        self.maxEpochs = maxEpochs
        self.minEpoch = math.ceil(maxEpochs * MIN_EPOCH_FRACTION)  # earliest epoch at which a stop is allowed
        self.escaped = False        # has the model left the all-blank phase
        self.bestEpoch = None       # lowest CER after escaping; this epoch is published
        self.bestCER = None
        self.referenceCER = None    # CER of the last real gain; patience is measured from it
        self.wait = 0
        self.failed = False
        self.reason = None

    def update(self, epoch, globalStep, result):
        """result is the output of validation.runValidation.  Returns True when training should stop."""
        metric = result[STOP_METRIC] if result[STOP_METRIC]["num_samples"] > 0 else result["all"]
        cer = metric["cer"]
        if not self.escaped:
            if metric["empty_fraction"] < MAX_EMPTY_FRACTION:
                self.escaped = True
            elif globalStep >= BLANK_FAIL_UPDATES or epoch >= self.maxEpochs:
                self.failed = True
                self.reason = (f"Model never left the all-blank phase: {metric['empty_fraction']:.0%} of "
                               f"validation transcripts are empty after {globalStep} updates")
                return True
            else:
                return False
        if self.bestCER is None or cer < self.bestCER:
            self.bestCER = cer
            self.bestEpoch = epoch
        if self.referenceCER is None or cer < self.referenceCER * (1.0 - MIN_RELATIVE_GAIN):
            self.referenceCER = cer
            self.wait = 0
        else:
            self.wait += 1
        if self.wait >= PATIENCE and epoch >= self.minEpoch:
            self.reason = (f"No {MIN_RELATIVE_GAIN:.1%} gain in {STOP_METRIC} CER for {PATIENCE} epochs; "
                           f"best CER {self.bestCER} at epoch {self.bestEpoch}")
            return True
        if epoch >= self.maxEpochs:
            self.reason = f"Reached the maximum of {self.maxEpochs} epochs; best CER {self.bestCER} at epoch {self.bestEpoch}"
            return True
        return False

    def settings(self):
        return {"stop_metric": STOP_METRIC, "patience": PATIENCE, "min_relative_gain": MIN_RELATIVE_GAIN,
                "min_epoch_fraction": MIN_EPOCH_FRACTION, "min_epoch": self.minEpoch,
                "max_empty_fraction": MAX_EMPTY_FRACTION, "blank_fail_updates": BLANK_FAIL_UPDATES}
