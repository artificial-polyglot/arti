package precheck

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/artificial-polyglot/arti/generic"
	log "github.com/artificial-polyglot/arti/logger"
	"github.com/artificial-polyglot/arti/request"
)

func TestValidateFilesWASM(t *testing.T) {
	var input = []string{
		"Chakma N2CCPBBS/.DS_Store",
		"Chakma N2CCPBBS/Text Files/Chakma Possible Discrepancies.docx",
		"Chakma N2CCPBBS/Text Files/N2CCPBBS Chakma Transliteration.pdf",
		"Chakma N2CCPBBS/Text Files/N2CCPBBS_audio_compare.html",
		"Chakma N2CCPBBS/Text Files/Chakma_N2CCPBBS_VR_Script.xlsx",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_219_JMS_002_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_192_1TI_002_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_236_2JN_001_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_234_1JN_004_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_037_MRK_009_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_089_JHN_021_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_205_HEB_001_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_148_1CO_015_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_013_MAT_013_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_142_1CO_009_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_172_EPH_004_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_016_MAT_016_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_231_1JN_001_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_038_MRK_010_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_180_COL_002_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_063_LUK_019_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_057_LUK_013_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_007_MAT_007_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_178_PHP_004_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_029_MRK_001_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_034_MRK_006_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_228_2PE_001_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_110_ACT_021_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_093_ACT_004_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_046_LUK_002_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_165_GAL_003_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_204_PHM_001_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_238_JUD_001_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_060_LUK_016_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_094_ACT_005_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_230_2PE_003_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_106_ACT_017_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_031_MRK_003_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_002_MAT_002_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_214_HEB_010_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_203_TTS_003_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_229_2PE_002_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_210_HEB_006_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_166_GAL_004_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_111_ACT_022_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_109_ACT_020_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_095_ACT_006_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_163_GAL_001_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_208_HEB_004_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_259_REV_021_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_147_1CO_014_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_032_MRK_004_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_099_ACT_010_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_218_JMS_001_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_022_MAT_022_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_213_HEB_009_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_149_1CO_016_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_179_COL_001_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_121_ROM_004_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_169_EPH_001_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_174_EPH_006_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_171_EPH_003_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_152_2CO_003_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_239_REV_001_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_175_PHP_001_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_066_LUK_022_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_168_GAL_006_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_027_MAT_027_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_136_1CO_003_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_117_ACT_028_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_113_ACT_024_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_098_ACT_009_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_164_GAL_002_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_047_LUK_003_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_090_ACT_001_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_140_1CO_007_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_160_2CO_011_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_006_MAT_006_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_035_MRK_007_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_102_ACT_013_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_030_MRK_002_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_215_HEB_011_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_003_MAT_003_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_145_1CO_012_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_125_ROM_008_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_189_2TH_002_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_061_LUK_017_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_114_ACT_025_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_235_1JN_005_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_088_JHN_020_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_036_MRK_008_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_012_MAT_012_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_193_1TI_003_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_056_LUK_012_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_181_COL_003_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_062_LUK_018_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_221_JMS_004_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_196_1TI_006_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_173_EPH_005_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_017_MAT_017_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_039_MRK_011_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_118_ROM_001_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_156_2CO_007_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_212_HEB_008_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_023_MAT_023_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_097_ACT_008_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_026_MAT_026_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_182_COL_004_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_119_ROM_002_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_067_LUK_023_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_170_EPH_002_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_167_GAL_005_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_091_ACT_002_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_161_2CO_012_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_211_HEB_007_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_202_TTS_002_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_190_2TH_003_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_143_1CO_010_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_258_REV_020_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_188_2TH_001_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_033_MRK_005_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_144_1CO_011_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_209_HEB_005_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_131_ROM_014_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_115_ACT_026_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_075_JHN_007_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_105_ACT_016_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_130_ROM_013_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_248_REV_010_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_255_REV_017_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_250_REV_012_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_058_LUK_014_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_052_LUK_008_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_247_REV_009_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_199_2TI_003_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_201_TTS_001_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_128_ROM_011_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_070_JHN_002_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_237_3JN_001_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_025_MAT_025_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_256_REV_018_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_049_LUK_005_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_241_REV_003_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_083_JHN_015_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_043_MRK_015_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_134_1CO_001_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_124_ROM_007_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_150_2CO_001_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_225_1PE_003_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_197_2TI_001_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_158_2CO_009_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_076_JHN_008_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_064_LUK_020_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_177_PHP_003_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_079_JHN_011_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_141_1CO_008_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_123_ROM_006_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_187_1TH_005_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_028_MAT_028_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_051_LUK_007_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_244_REV_006_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_020_MAT_020_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_040_MRK_012_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_080_JHN_012_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_135_1CO_002_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_242_REV_004_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_055_LUK_011_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_220_JMS_003_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_014_MAT_014_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_233_1JN_003_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_127_ROM_010_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_151_2CO_002_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_195_1TI_005_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_226_1PE_004_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_223_1PE_001_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_019_MAT_019_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_184_1TH_002_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_122_ROM_005_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_011_MAT_011_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_207_HEB_003_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_085_JHN_017_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_008_MAT_008_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_104_ACT_015_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_216_HEB_012_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_096_ACT_007_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_200_2TI_004_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_112_ACT_023_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_086_JHN_018_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_253_REV_015_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_005_MAT_005_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_073_JHN_005_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_157_2CO_008_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_129_ROM_012_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_103_ACT_014_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_107_ACT_018_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_224_1PE_002_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_065_LUK_021_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_077_JHN_009_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_153_2CO_004_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_176_PHP_002_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_183_1TH_001_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_222_JMS_005_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_257_REV_019_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_024_MAT_024_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_137_1CO_004_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_240_REV_002_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_048_LUK_004_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_042_MRK_014_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_082_JHN_014_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_245_REV_007_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_050_LUK_006_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_021_MAT_021_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_154_2CO_005_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_186_1TH_004_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_120_ROM_003_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_078_JHN_010_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_254_REV_016_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_249_REV_011_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_126_ROM_009_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_074_JHN_006_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_146_1CO_013_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_069_JHN_001_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_133_ROM_016_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_101_ACT_012_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_071_JHN_003_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_198_2TI_002_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_139_1CO_006_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_059_LUK_015_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_251_REV_013_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_246_REV_008_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_053_LUK_009_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_116_ACT_027_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_132_ROM_015_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_001_MAT_001_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_217_HEB_013_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_009_MAT_009_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_159_2CO_010_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_072_JHN_004_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_108_ACT_019_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_068_LUK_024_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_260_REV_022_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_004_MAT_004_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_162_2CO_013_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_100_ACT_011_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_045_LUK_001_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_092_ACT_003_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_087_JHN_019_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_138_1CO_005_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_252_REV_014_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_227_1PE_005_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_194_1TI_004_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_015_MAT_015_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_232_1JN_002_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_054_LUK_010_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_081_JHN_013_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_041_MRK_013_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_243_REV_005_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_084_JHN_016_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_044_MRK_016_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_185_1TH_003_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_206_HEB_002_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_010_MAT_010_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_191_1TI_001_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_155_2CO_006_VOX.mp3",
		"Chakma N2CCPBBS/N2CCPBBS Chapter mp3/N2_CCP_BBS_018_MAT_018_VOX.mp3",
		"Chakma N2CCPBBS/Text Files/SFM Text/46ROMCCPaux.SFM",
		"Chakma N2CCPBBS/Text Files/SFM Text/66JUDCCPaux.SFM",
		"Chakma N2CCPBBS/Text Files/SFM Text/57TITCCPaux.SFM",
		"Chakma N2CCPBBS/Text Files/SFM Text/611PECCPaux.SFM",
		"Chakma N2CCPBBS/Text Files/SFM Text/45ACTCCPaux.SFM",
		"Chakma N2CCPBBS/Text Files/SFM Text/42MRKCCPaux.SFM",
		"Chakma N2CCPBBS/Text Files/SFM Text/542THCCPaux.SFM",
		"Chakma N2CCPBBS/Text Files/SFM Text/50EPHCCPaux.SFM",
		"Chakma N2CCPBBS/Text Files/SFM Text/642JNCCPaux.SFM",
		"Chakma N2CCPBBS/Text Files/SFM Text/551TICCPaux.SFM",
		"Chakma N2CCPBBS/Text Files/SFM Text/49GALCCPaux.SFM",
		"Chakma N2CCPBBS/Text Files/SFM Text/67REVCCPaux.SFM",
		"Chakma N2CCPBBS/Text Files/SFM Text/631JNCCPaux.SFM",
		"Chakma N2CCPBBS/Text Files/SFM Text/531THCCPaux.SFM",
		"Chakma N2CCPBBS/Text Files/SFM Text/482COCCPaux.SFM",
		"Chakma N2CCPBBS/Text Files/SFM Text/562TICCPaux.SFM",
		"Chakma N2CCPBBS/Text Files/SFM Text/52COLCCPaux.SFM",
		"Chakma N2CCPBBS/Text Files/SFM Text/471COCCPaux.SFM",
		"Chakma N2CCPBBS/Text Files/SFM Text/653JNCCPaux.SFM",
		"Chakma N2CCPBBS/Text Files/SFM Text/51PHPCCPaux.SFM",
		"Chakma N2CCPBBS/Text Files/SFM Text/41MATCCPaux.SFM",
		"Chakma N2CCPBBS/Text Files/SFM Text/44JHNCCPaux.SFM",
		"Chakma N2CCPBBS/Text Files/SFM Text/59HEBCCPaux.SFM",
		"Chakma N2CCPBBS/Text Files/SFM Text/622PECCPaux.SFM",
		"Chakma N2CCPBBS/Text Files/SFM Text/60JASCCPaux.SFM",
		"Chakma N2CCPBBS/Text Files/SFM Text/43LUKCCPaux.SFM",
		"Chakma N2CCPBBS/Text Files/SFM Text/58PHMCCPaux.SFM"}

	var req request.Request
	req.Testament.OT = true
	req.Testament.NT = true
	errs := ValidateFilesWASM(&req, strings.Join(input, "\x00"), true)
	if len(errs) > 0 {
		t.Fatal(strings.Join(errs, "\n"))
	}
	wantAudio := "s3://arti-input/Chakma N2CCPBBS/N2CCPBBS Chapter mp3/*.mp3"
	if req.AudioData.AWSS3 != wantAudio {
		t.Errorf("AudioData.AWSS3 = %q, want %q", req.AudioData.AWSS3, wantAudio)
	}
	wantText := "s3://arti-input/Chakma N2CCPBBS/Text Files/SFM Text/*.SFM"
	if req.TextData.AWSS3 != wantText {
		t.Errorf("TextData.AWSS3 = %q, want %q", req.TextData.AWSS3, wantText)
	}
}

func TestUtility_validateBookId(t *testing.T) {
	ctx := context.Background()
	bookId, status := validateBookId(ctx, "TTL")
	if status != nil {
		t.Error(status)
	}
	if bookId != "TIT" {
		t.Error(bookId, "should have been revised to TIT")
	}
}

func TestUtility_parseFilenames(t *testing.T) {
	ctx := context.Background()
	test1 := generic.InputFile{MediaType: request.TextUSXEdit, Directory: "/ABC/DEF", Filename: "001GEN.usx"}
	status := parseFilenames(ctx, &test1)
	if status != nil {
		t.Error(status)
	}
	if test1.MediaId != "DEF" {
		t.Error("Media ID should be DEF")
	}
	if test1.BookId != "GEN" {
		t.Error("Book ID should be GEN")
	}
	if test1.BookSeq != "001" && test1.BookSeq != "1" {
		t.Error("Book Seq should be 001")
	}
	test2 := generic.InputFile{MediaType: request.TextUSXEdit, Directory: "/ABC/DEF", Filename: "GEN.usx"}
	status = parseFilenames(ctx, &test2)
	if status != nil {
		t.Error(status)
	}
	if test2.BookId != "GEN" {
		t.Error("Book ID should be GEN")
	}
	if test2.BookSeq != "1" {
		t.Error("Book Seq should be 1")
	}
	test3 := generic.InputFile{MediaType: request.TextUSXEdit, Directory: "/ABC/DEF", Filename: "1GEN.usx"}
	status = parseFilenames(ctx, &test3)
	if test3.BookId != "GEN" {
		t.Error("For 1GEN.usx  book_id=GEN")
	}
	if test3.BookSeq != "1" {
		t.Error("For 1GEN.usx  bookSeq should be 1")
	}
}

func TestUtility_FindTextBookId(t *testing.T) {
	bookId := findTextBookId("0231TITXX.sfm")
	if bookId != "TIT" {
		t.Error("For 0231TITXX.sfm bookId = TIT")
	}
	bookId = findTextBookId("0231TIXX.sfm")
	if bookId != "1TI" {
		t.Error("For 0231TIXX.sfm bookId = 1TI")
	}
	bookId = findTextBookId("01231TIT.sfm")
	if bookId != "TIT" {
		t.Error("For 0231TIT.sfm bookId = TIT")
	}
	bookId = findTextBookId("01231TI.sfm")
	if bookId != "1TI" {
		t.Error("For 01231TI.sfm bookId = 1TI")
	}
	fmt.Println(bookId)
}

func TestUtility_FindAudioBookId(t *testing.T) {
	bookId, chapter, err := findAudioBookId(strings.Split("N2_QAE_BSP_006_MAT_006_VOX.mp3", "_"))
	if bookId != "MAT" {
		t.Error("For N2_QAE_BSP_006_MAT_006_VOX.mp3 bookId = MAT")
	}
	if chapter != 6 {
		t.Error("For N2_QAE_BSP_006_MAT_006_VOX.mp3 chapter = 6")
	}
	if err != nil {
		t.Error("For N2_QAE_BSP_006_MAT_006_VOX.mp3 err = nil")
	}
}

func TestParseV4AudioFilename(t *testing.T) {
	ctx := context.Background()
	var status *log.Status
	var file generic.InputFile
	file.Filename = `ENGESVN2DA_B001_MAT_001.mp3`
	status = parseV4AudioFilename(ctx, &file)
	if status != nil {
		t.Error(status)
	}
	if file.MediaId != `ENGESVN2DA` {
		t.Error(`mediaId is incorrect`, file.MediaId)
	}
	if file.Testament != `NT` {
		t.Error(`Testament is incorrect`, file.Testament)
	}
	if file.BookId != `MAT` {
		t.Error(`BookId is incorrect`, file.BookId)
	}
	if file.BookSeq != `001` {
		t.Error(`BookSeq is incorrect`, file.BookSeq)
	}
	if file.Chapter != 1 {
		t.Error(`Chapter is incorrect`, file.Chapter)
	}
	if file.Verse != `` {
		t.Error(`Verse is incorrect`, file.Verse)
	}
	//fmt.Println("File", file)
	var file2 generic.InputFile
	file2.Filename = `IRUNLCP1DA_B013_1TH_001_001-001_010.mp3`
	status = parseV4AudioFilename(ctx, &file2)
	if status != nil {
		t.Error(status)
	}
	if file2.Verse != `001` {
		t.Error(`Verse is incorrect`, file2.Verse)
	}
	if file2.ChapterEnd != 1 {
		t.Error(`ChapterEnd is incorrect`, file2.ChapterEnd)
	}
	if file2.VerseEnd != `010` {
		t.Error(`VerseEnd is incorrect`, file2.VerseEnd)
	}
	//fmt.Println("File2", file2)
}
