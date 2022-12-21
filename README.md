# Stable Diffusion Discord Bot

This is a Discord bot that interfaces with the Automatic1111 API, from this project: https://github.com/AUTOMATIC1111/stable-diffusion-webui 

## Installation

1. Clone this repository
2. Install Go
3. Build the bot with `go build`

## Usage

1. Create a Discord bot and get the token
2. Run the bot with `./stable-diffusion-discord-bot -token <token> -guild <guild ID> -host <webui host, e.g. http://127.0.0.1:7860>`

## Commands

- `/imagine <text>` - Creates an image with the text

## How it Works

The bot implements a FIFO queue (first in, first out). When a user issues the `/imagine` command, their prompt is added to the end of the queue.

The bot then checks the queue every second. If the queue is not empty, and there is no image currently being processed, it will send the first prompt to the webui, and then remove it from the queue.

After the webui has finished processing the prompt, the bot will then update the reply message with the finished image.

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.