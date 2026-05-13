TODO_HOME := $(or $(TODO_HOME),$(HOME)/.todo)
PIDFILE := $(TODO_HOME)/todo.pid
LOGFILE := $(TODO_HOME)/todo.log
ERRLOGFILE := $(TODO_HOME)/todo_err.log

.PHONY: start stop restart

build:
	go build .

start: build
	@if [ -f $(PIDFILE) ] && kill -0 `cat $(PIDFILE)` 2>/dev/null; then \
		echo "todo server is already running (PID $$(cat $(PIDFILE)))"; \
	else \
		./todo serve >> $(LOGFILE) 2>>$(ERRLOGFILE) & \
		echo $$! > $(PIDFILE); \
		echo "todo server started (PID $$(cat $(PIDFILE)))" >> $(LOGFILE); \
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
