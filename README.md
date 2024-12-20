# flash

`flash` is a small script which removes some of the friction with flashing new firmware onto a [Glove80 split ergonomic keyboard](https://www.moergo.com/).

I was annoyed by having to push changes to my config, go to GitHub, download the `uf2` file from the latest build, and copy it to each glove. Especially when I would be tweaking little aspects of the config on a regular basis while finding a config that worked for me.

This script will:

1. Verify that both gloves are connected to your machine
2. Download the most recent `uf2` artifact generated by the GitHub actions of your [Glove80 ZMK config repo](https://github.com/moergo-sc/glove80-zmk-config)
3. Copy the `uf2` file to each glove

To run the script:

1. From the root directory, run `go build`
2. Run the script with `./flash`

You may also provide the `--directory`/`-d` option which will specify the path of the parent directory where the volumes for each of your gloves will reside. By default this is `/Volumes` for MacOS.
