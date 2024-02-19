package main

// Convert pages to strings otherwise the render() function won't work for both generated and templated html

import "embed"

//go:embed templates/index.html
var hindexpage string

//go:embed templates/auth.html
var hauthpage string

//go:embed static
var hcss embed.FS
