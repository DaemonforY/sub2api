# Exports Real-ESRGAN's realesr-general-x4v3 (SRVGGNetCompact, BSD-3-Clause) to ONNX.
import sys

import torch
import torch.nn as nn
import torch.nn.functional as F


class SRVGGNetCompact(nn.Module):
    def __init__(self, num_feat=64, num_conv=32, upscale=4):
        super().__init__()
        self.upscale = upscale
        self.body = nn.ModuleList([nn.Conv2d(3, num_feat, 3, 1, 1), nn.PReLU(num_parameters=num_feat)])
        for _ in range(num_conv):
            self.body.append(nn.Conv2d(num_feat, num_feat, 3, 1, 1))
            self.body.append(nn.PReLU(num_parameters=num_feat))
        self.body.append(nn.Conv2d(num_feat, 3 * upscale * upscale, 3, 1, 1))
        self.upsampler = nn.PixelShuffle(upscale)

    def forward(self, x):
        out = x
        for layer in self.body:
            out = layer(out)
        return self.upsampler(out) + F.interpolate(x, scale_factor=self.upscale, mode="nearest")


src, dst = sys.argv[1], sys.argv[2]
model = SRVGGNetCompact()
state = torch.load(src, map_location="cpu", weights_only=True)
model.load_state_dict(state.get("params", state))
model.eval()
torch.onnx.export(model, torch.rand(1, 3, 64, 64), dst, input_names=["input"], output_names=["output"],
                  dynamic_axes={"input": {2: "h", 3: "w"}, "output": {2: "h4", 3: "w4"}}, opset_version=17, dynamo=False)
