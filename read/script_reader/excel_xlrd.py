import xlrd
import json
import sys

wb = xlrd.open_workbook(sys.argv[1])
ws = wb.sheet_by_index(0)
rows = []
for i in range(ws.nrows):
    row = [str(ws.cell_value(i, j)) for j in range(ws.ncols)]
    if any(v.strip() for v in row):
        rows.append(row)
print(json.dumps(rows))

# python excel_xlrd.py ./test_data/CORE_Scr_1065p_1Eng__14_Spkr_Tajik_N2_TGK_IBT.xls