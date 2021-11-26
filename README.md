# google-rotation-planner

Plan rota on google calendar.

## Usage

Get a `credentials.json` file from GCloud and place it in the same folder as the executable.

Place your configuration file in `config.yaml`:

```yaml
# duration of a shift in weeks (integer)
shiftDuration: 1

# start date for the rotation (default to time.Now())
startDate: 2021-11-26

# the calendar ID where to configure the rotation
calendarID: <calendar-id>

# the members of the rotation
members:
  - name: John Doe
    email: jdoe@example.com
  - name: Foo Bar
    email: foo.bar@example.com
  - name: Jean Dupond
    email: dupond@example.com
```
