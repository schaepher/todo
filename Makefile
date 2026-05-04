TODO_HOME := $(or $(TODO_HOME),$(HOME)/.todo)
PIDFILE := $(TODO_HOME)/todo.pid

.PHONY: start stop restart

start:
	@if [ -f $(PIDFILE) ] && kill -0 `cat $(PIDFILE)` 2>/dev/null; then \
		echo "todo server is already running (PID $$(cat $(PIDFILE)))"; \
	else \
		./todo serve & \
		echo $$! > $(PIDFILE); \
		echo "todo server started (PID $$(cat $(PIDFILE)))"; \
	fi

stop:
	@if [ -f $(PIDFILE) ] && kill -0 `cat $(PIDFILE)` 2>/dev/null; then \
		echo "stopping todo server (PID $$(cat $(PIDFILE)))"; \
		kill `cat $(PIDFILE)`; \
		rm -f $(PIDFILE); \
		echo "stopped"; \
	else \
		echo "todo server is not running"; \
		rm -f $(PIDFILE); \
	fi

restart: stop start
