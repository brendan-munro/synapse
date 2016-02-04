# synapse

Synapse proves a simple context based http.Handler and chaining tool.

### Understanding

Synapse was written to have a simple http.Handler that passes along a context.Context with each request.

Many other solutions already exists but they tend to either implement their own context object or have everything wrapped up behind a customer mux.

### Usage

An example of using a middleware chain with a handler can be found in the tests.

### Credits

github.com/justinas/alice Was used as a base when making the chaining mechanism

### Contributing/Bugs

If you happen to come across a bug or can think of a better feature implementation please open a new appropriate issue and submit a pull request (if relevant)/
