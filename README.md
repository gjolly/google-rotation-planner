# google-rotation-planner

This tool allows you to easily configure your on-call rotations on Google Calendar.

## Installation

The recommanded way to install this tool is to download the right pre-built binary for your system from the [latest release](https://github.com/gjolly/google-rotation-planner/releases/latest).

You can also install the tool with `golang`:

```bash
go install github.com/gjolly/google-rotation-planner@latest
```

## Usage

Get a `credentials.json` file from GCloud and place it in `~/.google-rotation-planner/credentials.json` (see instructions [here](https://developers.google.com/calendar/api/quickstart/go#set_up_your_environment) to get started).

Place your configuration file in `config.yaml`:

```yaml
# duration of a shift in weeks (integer)
shiftDuration: 1

# start date for the rotation (default to time.Now())
startDate: 2021-11-26

# the calendar ID where to configure the rotation
calendarID: <calendar-id>

# optional list of attachments
attachments:
  - name: Notes
    url: https://docs.google.com/document/d/xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx/edit?usp=sharing

# the members of the rotation
members:
  - name: John Doe
    email: jdoe@example.com
  - name: Foo Bar
    email: foo.bar@example.com
  - name: Jean Dupond
    email: dupond@example.com
```

Then, run the tool:

```bash
❯ ./google-rotation-planner
creating shift for Test User 1, first shift starting on 2024-11-03 00:00:00 +0000 UTC and finshing on 2024-11-10 00:00:00 +0000 UTC (1 week(s))
creating shift for Test User 2, first shift starting on 2024-11-10 00:00:00 +0000 UTC and finshing on 2024-11-17 00:00:00 +0000 UTC (1 week(s))
```

You can remove all the events from the calendar using the `cleanup` command:

```bash
❯ ./google-rotation-planner -cleanup
deleting  Test User 1 on duty
deleting  Test User 2 on duty
```

To add new members to the rotation, cleanup the existing events, add the new member to the configuration file and re-run `google-rotation-planner`.

## Build

Binaries can be downloaded from here: https://github.com/gjolly/google-rotation-planner/releases/latest

To build it yourself:

```bash
go build -o . ./...
```
