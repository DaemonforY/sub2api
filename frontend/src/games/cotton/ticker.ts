// Keeps an online host's game running while its tab is in the background, where the page's
// own animation frames stop. Posts a tick roughly every frame.
setInterval(() => postMessage(0), 16)
