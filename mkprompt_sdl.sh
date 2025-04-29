#!/bin/sh

source ~/personal/.shhelpers

files_for_llm `find . | grep -v "\.sh" | grep -v attic | grep -v mkprompt | grep sdl `
