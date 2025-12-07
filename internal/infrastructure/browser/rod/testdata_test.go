package rod

// TestHTML templates for testing
const (
	BasicHTML = `<!DOCTYPE html>
<html>
<head><title>Test Page</title></head>
<body>
	<h1>Hello World</h1>
</body>
</html>`

	FormHTML = `<!DOCTYPE html>
<html>
<body>
	<form id="testForm">
		<input id="username" type="text" name="username" />
		<input id="password" type="password" name="password" />
		<button id="submit" type="submit">Submit</button>
	</form>
</body>
</html>`

	InteractiveHTML = `<!DOCTYPE html>
<html>
<body>
	<button id="btn">Click Me</button>
	<div id="result"></div>
	<script>
		document.getElementById('btn').addEventListener('click', function() {
			document.getElementById('result').textContent = 'Clicked!';
		});
	</script>
</body>
</html>`

	RichUIHTML = `<!DOCTYPE html>
<html>
<body>
	<button id="btn1" aria-label="First Button">Button 1</button>
	<button id="btn2" role="button">Button 2</button>
	<input id="input1" type="text" placeholder="Enter text" />
	<textarea id="textarea1"></textarea>
	<a href="/page1" id="link1">Link 1</a>
	<a href="/page2" id="link2" aria-label="Second Link">Link 2</a>
	<div role="button" id="divBtn" data-tooltip="Tooltip">Div Button</div>
</body>
</html>`

	ScrollableHTML = `<!DOCTYPE html>
<html>
<body style="height: 5000px;">
	<h1 id="top">Top of Page</h1>
	<div style="margin-top: 2000px;" id="middle">Middle</div>
	<div style="margin-top: 2000px;" id="bottom">Bottom</div>
</body>
</html>`
)
