# vscode-modernizer-extension README

A VSCode extension for displaying and ranking most frequently asked prompts

## Features

- Generate a Prompt with a random instruct and display the response
- Up- or Downvote the Response
- Display the current Promptcount
- Retrieve a random or the highest-ranked* prompt
- Save custom instructs locally
- find semantically similar functions via vector comparison

## Requirements

- The modernizer server must be running and be accessible
- Behind that a weaviate, redis and ollama instance are necessary

## Extension Settings


## Known Issues


## Release Notes

### 1.1.0

- add functionality to show next responses in output window
- display link to exact code reference used in the original response
- add commands to context menu and command palette

### 1.0.0
- First major release with most functional requirements implemented

### 0.0.1

- Initial release 
