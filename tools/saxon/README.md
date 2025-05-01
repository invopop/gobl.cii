# Saxon-HE Docker Image
This repository contains a Docker image for running Saxon-HE (a free version of the Saxon XSLT and XQuery processor) via a simple Docker container. It simplifies the process of using Saxon-HE for transforming XML documents with XSLT stylesheets, without needing to install Java or Saxon-HE locally.

## Usage
You can run the Docker container with the following basic command:

```bash
docker run --rm -v "$PWD" -w /data saxon -s:/path/to/input.xml -xsl:/path/to/transform.xsl -o:/path/to/output.xml
```
Parameters 
- input.xml: The XML file you want to transform.
- transform.xsl: The XSLT file to apply to the XML file.
- output.xml: The output file where the transformed XML will be written to. This is optional.

The `--rm` flag ensures that the container is removed after the command completes.

## Calling the Docker Image from Go
You can also call this Docker image programmatically from Go using the exec package. Here's an example Go code snippet that captures the output of the command instead of passing in output.xml:

```go
package main

import (
	"bytes"
	"fmt"
	"os/exec"
)

func main() {
	// Define the Docker command with parameters
	cmd := exec.Command(
		"docker", "run", "--rm",
		"-v", "/path/to/xml:/data", // Adjust path as needed
		"-w", "/data",
		"saxon-he",
		"-s:input.xml", "-xsl:transform.xsl",
	)

	// Create buffers to capture stdout and stderr
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	// Execute the command
	err := cmd.Run()
	if err != nil {
		// Handle errors, print stderr if available
		fmt.Printf("Error running command: %v\n", err)
		fmt.Println("Stderr:", errOut.String())
		return
	}

	// Print the standard output captured from the command
	fmt.Println("Output:", out.String())
}
```