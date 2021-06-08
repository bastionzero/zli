# zli

## Bastionzero

Bastionzero is a simple to use zero trust access SaaS for dynamic cloud environments. Bastionzero is the most secure way to lock down remote access to servers, containers, clusters, and VM’s in any cloud, public or private. For more information go to [Bastionzero](https://www.bastionzero.com).

The zli is a cli client for interacting with the Bastionzero SaaS

## Install

```
npm i -g @bastionzero/zli
```

## Developer processes

### Build and test
```
npm run start -- <cmd> [args] --flag flagArg
```

### Run against stage or dev
The following command is hidden from the help menu:
```
npm run start -- <cmd> [args] --configName <prod | stage | dev>
zli --configName <prod | stage | dev>
```

 ## Release Process

 We use [pkg](https://github.com/vercel/pkg) to package the node.js application into a single executable that can be run even without node or any npm dependencies are installed. The target executables can be configured in the `package.json` file for different OSs as documented [here](https://github.com/vercel/pkg#targets) but the default is to build windows, mac, and linux executable for the current node.js version and arch. Use `npm run release` to package the app and output executables to a `bin` directory.

### Release Versioning

The executables will be published to the s3 bucket with 2 different path prefixes each time the codebuild job is run:

1. `webshell-cli-release/release/latest/`

2. `webshell-cli-release/release/{version}`

Where {version} is the version that is defined in the `package.json` file. This means older versions are still accessible but the `latest` folder will always overwritten by the codebuild job.

## Installing a release

The latest releases can be found here:
```
Mac:        download-zli.bastionzero.com/release/latest/bin/zli-macos
Linux:      download-zli.bastionzero.com/release/latest/bin/zli-linux
Windows:    download-zli.bastionzero.com/release/latest/bin/zli-win.exe
```

### Mac users:
 - download the executable
 - `chmod +x` the executable
 - Run the program once and see a warning from Apple
 - Go to `System Preferences > Security & Privacy > General > Allow zli`
 - Run the executable again and confirm for Apple

Minor releases generating warnings for users to update their zli. Major releases
will cause all lower major versions to error.

### Linux users:
 - download the executable
 - `chmod +x` the executable

### Windows users:
 - Run the executable from the command line, even though you can technically click it

## Running zli

```
zli help  # auto-gen help menu
```
