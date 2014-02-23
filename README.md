Current Status: UNSTABLE
======
Very unstable proof-of-concept. No tests!


fsproxy
=======

A FUSE-based filesystem proxy with hooks!

This lets you (re)mount a portion of the filesystem elsewhere while hooking in to listing and reading functionality so you can add/remove/alter files and directories.

An example of using the hooks to uppercase the contents of all the files, check out `uppercasefs`.

