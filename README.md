*goaidifftool*

This tool allows you to provide a prompt and a text and outputs a diff between the input text and the AI-generated text. It is designed for writers and editors to manage editing large bodies of AI-edited text.

Users just have to clicked the colored text to remove it.

The most of the code is just rehashed from [AgentSmithU](https://github.com/mattyleecifer/AgentSmithU). The diff function is from [this diff tool](https://github.com/sergi/go-diff/diffmatchpatch) - I reworked the pretty print function to output HTML tags instead of color tags. The new function is very professionally placed in htmldiff.txt.   

Screenshot:
![Screenshot](https://github.com/mattyleecifer/goaidifftool/blob/main/examples/demo.png)
