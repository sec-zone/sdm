# Simple Download Manager (SDM)

Welcome to the Simple Download Manager (SDM) project! This command-line tool is built with Go and leverages the TUI library to provide a user-friendly interface for managing downloads. SDM is designed to be lightweight, efficient, and easy to use, with key features including:

- **Pause/Resume**: Gain control over your downloads and manage your bandwidth effectively.
- **Multi-Connection Download**: Accelerate your downloads by splitting files into multiple parts and downloading them concurrently.

## Getting Started

To get started with SDM, please follow the instructions below:

1. Ensure you have Go(version 1.18 and upper) installed on your system.
2. Clone the repository to your local machine.
3. Navigate to the cloned directory and build the project using `go build -ldflags="-w -s -buildid=" -trimpath -o sdm.exe`.

## Usage
Usage of sdm.exe:

- -H string: 
    Specify the custom header you want sent when downloading. Example: x-authorization-header=abcd;x-test-header=foo
- -c int:
        Number of connections to download (default 10)
- -o string:
        Specify the file name of downloaded file. If not specified the program try to get filename from content-disposition header.
-   -r int:
        Specify the number of times to retry downloading if error is encountered. (default 10)
- -u string:
        Specify the target URL for downloading file.

## Contributing

We encourage contributions from the community! Whether you're fixing bugs, adding new features, or improving documentation, your help is welcome. Please read through our contributing guidelines before submitting your pull request.

Thank you for supporting SDM and open-source software!

--
