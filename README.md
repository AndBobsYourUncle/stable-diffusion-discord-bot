# Stable Diffusion Discord Bot

This is a Discord bot that interfaces with the Automatic1111 API, from this project: https://github.com/AUTOMATIC1111/stable-diffusion-webui 

## Installation

1. Clone this repository
2. Install Go
3. Build the bot with `go build`

## Usage

1. Create a Discord bot and get the token
2. Add the Discord bot to your Discord server. It needs permissions to post messages, use slash commands, mentioning anyone, and uploading files.
2. Run the bot with `./stable_diffusion_bot -token <token> -guild <guild ID> -host <webui host, e.g. http://127.0.0.1:7860>`

## Commands

- `/imagine <text>` - Creates an image with the text

## How it Works

The bot implements a FIFO queue (first in, first out). When a user issues the `/imagine` command, their prompt is added to the end of the queue.

The bot then checks the queue every second. If the queue is not empty, and there is no image currently being processed, it will send the first prompt to the webui, and then remove it from the queue.

After the webui has finished processing the prompt, the bot will then update the reply message with the finished image.

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

There are lots more features that could be added to this bot, such as:

- [ ] Moving defaults to a config file
- [x] Ability to easily re-roll an image
- [x] Generating multiple images at once
- [x] Ability to upscale the resulting images
- [ ] Ability to generate variations on a grid image
- [ ] Ability to tweak more settings when issuing the `/imagine` command
- [ ] Image to image processing

I'll probably be adding a few of these over time, but any contributions are also welcome.

## Why Go?

I like Go a lot better than Python, and for me it's a lot easier to maintain dependencies with Go modules versus running a bunch of different Anaconda environments.

It's also able to be cross-compiled to a wide range of platforms, which is nice.
