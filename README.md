# Stable Diffusion Discord Bot

This is a Discord bot that interfaces with the Automatic1111 API, from this project: https://github.com/AUTOMATIC1111/stable-diffusion-webui 

Video showing off the current features:
https://www.youtube.com/watch?v=of5MBh3ueMk

## Installation

1. Clone this repository
2. Install Go
3. Build the bot with `go build`

## Usage

1. Create a Discord bot and get the token
2. Add the Discord bot to your Discord server. It needs permissions to post messages, use slash commands, mentioning anyone, and uploading files.
3. Run the bot with `./stable_diffusion_bot -token <token> -guild <guild ID> -host <webui host, e.g. http://127.0.0.1:7860>`
4. The first run will generate a new SQLite DB file in the current working directory.

## Commands

### `/imagine`

Creates an image from a text prompt. (e.g. `/imagine cute kitten riding a skateboard`)

Available options:
- Aspect Ratio
  - `--ar <width>:<height>`
  - e.g. `/imagine cute kitten riding a skateboard --ar 16:9`

## How it Works

The bot implements a FIFO queue (first in, first out). When a user issues the `/imagine` command (or uses an interaction button), they are added to the end of the queue.

The bot then checks the queue every second. If the queue is not empty, and there is nothing currently being processed, it will send the top interaction to the Automatic1111 WebUI API, and then remove it from the queue.

After the Automatic1111 has finished processing the interaction, the bot will then update the reply message with the finished result.

Buttons are added to the Discord response message for interactions like re-roll, variations, and up-scaling.

All image generations are saved into a local SQLite database, so that the parameters of the image can be retrieved later for variations or up-scaling.

<img width="846" alt="Screenshot 2022-12-22 at 4 25 03 PM" src="https://user-images.githubusercontent.com/7525989/209247258-8c637265-b0b2-419a-98c6-95c4bb78504f.png">

<img width="667" alt="Screenshot 2022-12-22 at 4 25 18 PM" src="https://user-images.githubusercontent.com/7525989/209247280-4318a73a-71f4-48aa-8310-7fdfbbbf6820.png">

Options like aspect ratio are extracted and sanitized from the text prompt, and then the resulting options are stored in the database record for the image generation (for further variations or upscaling):

<img width="995" alt="Screenshot 2022-12-28 at 4 30 43 PM" src="https://user-images.githubusercontent.com/7525989/209888645-b616fbb1-955a-4d3e-9a25-ce43baa6cfbd.png">

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

There are lots more features that could be added to this bot, such as:

- [ ] Moving defaults to the database
- [ ] Per-user defaults/settings, as well as enforcing limits on a user's usage of the bot
- [x] Ability to easily re-roll an image
- [x] Generating multiple images at once
- [x] Ability to upscale the resulting images
- [x] Ability to generate variations on a grid image
- [ ] Ability to tweak more settings when issuing the `/imagine` command (like aspect ratio)
- [ ] Image to image processing

I'll probably be adding a few of these over time, but any contributions are also welcome.

## Why Go?

I like Go a lot better than Python, and for me it's a lot easier to maintain dependencies with Go modules versus running a bunch of different Anaconda environments.

It's also able to be cross-compiled to a wide range of platforms, which is nice.
