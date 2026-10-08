import torch


def ensureMinimumTensorSize(tensor, minTensorLength, padValue):
    originalLen = tensor.shape[-1]
    if originalLen < minTensorLength:
        paddingNeeded = minTensorLength - originalLen
        tensor = torch.nn.functional.pad(tensor, (0, paddingNeeded), mode='constant', value=padValue)
    mask = torch.ones(len(tensor), dtype=torch.int)
    if len(tensor) > originalLen:
        mask[originalLen:] = 0  # Mark padding as invalid
    return tensor, mask

