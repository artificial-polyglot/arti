# Decides after each epoch whether training should continue, cool down, or stop, using the
# validation CER.  NumEpochs from the request is the most epochs that will run, and the base
# linear learning rate schedule decays over all of them.  When the CER stops improving, the
# learning rate is cut to zero over COOLDOWN_EPOCHS more epochs so the model settles, and then
# training stops.  The published epoch is the best of those settled epochs, never one taken while
# the learning rate was still high.  These constants may later become request parameters.

STOP_METRIC = "all"         # validation group whose CER decides: "fa_score", "silence" or "all"
PATIENCE = 4                # start the cool-down after this many epochs in a row without a real gain
MIN_RELATIVE_GAIN = 0.005   # a gain is real when CER falls by at least 0.5% of the reference CER
COOLDOWN_EPOCHS = 2         # epochs over which the learning rate is cut to zero before stopping
MAX_EMPTY_FRACTION = 0.5    # while at least this fraction of transcripts is empty, the model has
                            # not yet left the all-blank phase, and patience is not counted
BLANK_FAIL_UPDATES = 4000   # training fails if it is still in the all-blank phase after this many updates

CONTINUE = "continue"
COOLDOWN = "cooldown"
STOP = "stop"


class EarlyStopper:
    def __init__(self, maxEpochs):
        self.maxEpochs = maxEpochs
        self.escaped = False        # has the model left the all-blank phase
        self.referenceCER = None    # CER of the last real gain; patience is measured from it
        self.wait = 0
        self.cooldownStart = None   # last epoch before the cool-down, once it has begun
        self.cooldownEnd = None     # epoch at which the learning rate reaches zero
        # Epochs from settledFrom on are candidates for publishing.  If the CER never levels off,
        # the base schedule's last epochs are the settled ones, since its rate also ends at zero.
        self.settledFrom = max(1, maxEpochs - COOLDOWN_EPOCHS + 1)
        self.cers = {}              # epoch -> CER, after escaping the all-blank phase
        self.failed = False
        self.reason = None

    @property
    def bestEpoch(self):
        """The settled epoch with the lowest CER so far; this is the one published."""
        settled = [e for e in self.cers if e >= self.settledFrom]
        return min(settled, key=lambda e: (self.cers[e], -e)) if settled else None

    @property
    def lowestEpoch(self):
        """The epoch with the lowest CER of any, settled or not; reported for comparison."""
        return min(self.cers, key=lambda e: (self.cers[e], -e)) if self.cers else None

    def update(self, epoch, globalStep, result):
        """result is the output of validation.runValidation.  Returns CONTINUE, COOLDOWN (the
        learning rate is to fall to zero by epoch self.cooldownEnd), or STOP."""
        metric = result[STOP_METRIC] if result[STOP_METRIC]["num_samples"] > 0 else result["all"]
        cer = metric["cer"]
        if not self.escaped:
            if metric["empty_fraction"] < MAX_EMPTY_FRACTION:
                self.escaped = True
            elif globalStep >= BLANK_FAIL_UPDATES or epoch >= self.maxEpochs:
                self.failed = True
                self.reason = (f"Model never left the all-blank phase: {metric['empty_fraction']:.0%} of "
                               f"validation transcripts are empty after {globalStep} updates")
                return STOP
            else:
                return CONTINUE
        self.cers[epoch] = cer
        if self.cooldownStart is not None:
            if epoch >= self.cooldownEnd:
                self.reason = (f"No {MIN_RELATIVE_GAIN:.1%} gain in {STOP_METRIC} CER for {PATIENCE} epochs "
                               f"after epoch {self.cooldownStart - PATIENCE}; cooled down over epochs "
                               f"{self.cooldownStart + 1}-{self.cooldownEnd}; " + self.bestText())
                return STOP
            return CONTINUE
        if epoch >= self.maxEpochs:
            self.reason = f"Reached the maximum of {self.maxEpochs} epochs; " + self.bestText()
            return STOP
        if self.referenceCER is None or cer < self.referenceCER * (1.0 - MIN_RELATIVE_GAIN):
            self.referenceCER = cer
            self.wait = 0
            return CONTINUE
        self.wait += 1
        if self.wait < PATIENCE:
            return CONTINUE
        self.cooldownStart = epoch
        self.cooldownEnd = min(epoch + COOLDOWN_EPOCHS, self.maxEpochs)
        self.settledFrom = min(self.settledFrom, epoch + 1)
        return COOLDOWN

    def bestText(self):
        text = f"published epoch {self.bestEpoch} CER {self.cers.get(self.bestEpoch)}"
        if self.lowestEpoch != self.bestEpoch:
            text += f"; lowest CER {self.cers[self.lowestEpoch]} at epoch {self.lowestEpoch}"
        return text

    def settings(self):
        return {"stop_metric": STOP_METRIC, "patience": PATIENCE, "min_relative_gain": MIN_RELATIVE_GAIN,
                "cooldown_epochs": COOLDOWN_EPOCHS, "cooldown_start": self.cooldownStart,
                "max_empty_fraction": MAX_EMPTY_FRACTION, "blank_fail_updates": BLANK_FAIL_UPDATES}


class LRPlan:
    """The learning rate multiplier for each update, for torch's LambdaLR.  It warms up linearly,
    then decays linearly to zero at totalSteps, the end of the last allowed epoch.  Once a cool-down
    begins, it falls linearly from its current value to zero at the cool-down's end instead."""

    def __init__(self, warmupSteps, totalSteps):
        self.warmupSteps = warmupSteps
        self.totalSteps = totalSteps
        self.cooldownStep = None
        self.cooldownEndStep = None
        self.cooldownFactor = None

    def baseFactor(self, step):
        if step < self.warmupSteps:
            return step / max(1, self.warmupSteps)
        return max(0.0, (self.totalSteps - step) / max(1, self.totalSteps - self.warmupSteps))

    def __call__(self, step):
        if self.cooldownStep is None:
            return self.baseFactor(step)
        remaining = self.cooldownEndStep - step
        return max(0.0, self.cooldownFactor * remaining / max(1, self.cooldownEndStep - self.cooldownStep))

    def startCooldown(self, step, endStep):
        """step is the number of updates done; the rate reaches zero after endStep updates."""
        self.cooldownFactor = self.baseFactor(step)
        self.cooldownStep = step
        self.cooldownEndStep = min(endStep, self.totalSteps)
