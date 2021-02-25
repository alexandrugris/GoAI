import matplotlib.pyplot as plt
import numpy as np

{{range .}} {{if eq .Type "line"}}

values = {{index .Values 0 | CompressByMean .Count | ToPythonArray}}
plt.plot(values)

{{end}} {{end}}

plt.show()