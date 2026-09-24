# Writing tickets and comments

A tester reads this a week from now. They have the build and a browser, no code, no logs. After reading they know what to click.

## Rules

1. Write the title the way the user would say it to support: what they did and what they saw. For a bug, product name first, then a colon. For a task, verb + object. No cause, no solution, no code names. Under 70 characters.
2. For a bug, put the version or build on the first line of the body, exactly as the source gives it. If the source does not give it, ask before writing. Tasks have no version line.
3. Write one to three sentences on what happened. Then, if the steps are not obvious, list them.
4. Every sentence is either a step or something you observed. Move any guess about the cause into one sentence that starts with "probably" or "seems like". Delete every other guess.
5. Text the user sees on screen goes in quotes: "Connecting to Home-5G". A code fence is only for log lines, error output and commands, pasted exactly. Never paraphrase them.
6. Name what the user sees instead of evaluating it: not "the schedule is unreliable", but "the 06:00 setpoint is skipped and the display still shows it as active".
7. Labels are plain words with a colon on their own line: Steps:, Expected:, Actual:, Workaround:, Request:. No headings, no bold, no bullet labels.
8. A task is "Currently X. Please Y." Name the screen or menu item. One sentence of why, only when it is not obvious. Dev-only tasks may contain code, payloads and paths.
9. One issue per ticket. If the notes contain two, write two tickets.
10. A question to the developer is fine. A routing note is fine ("if this is app-side, assign to Mobile").
11. Stay around 100 words for a bug and 40 for a task, not counting logs. If it does not fit, split the ticket.
12. Never invent a version, a build number, a name or an error text. Take them from the source or ask.
13. Before posting: delete the closing sentence; delete "not X but Y" and any three-item list you did not need; check every version and error text against the source.

## Comments

Dev to QA: one sentence on what changed, then the build to test on. QA: version on the first line, what you checked, then "Closing ticket" or "Still reproducible". Comment only when you add information. No progress diaries. To correct yourself, edit the original.

## Samples. Copy the form of the closest one and replace the content.

Bug, structured:
```
Summary: Hearth T2: active sensor resets to Remote after factory reset

2.8.20260914dev-build512
After factory resetting Hearth T2, the active sensor defaults to Remote instead of Built-in.

Steps:
Set Active Sensor to Built-in.
Factory Reset All Settings.
Check Active Sensor.

Expected: Built-in.
Actual: Remote.
```

Bug, prose:
```
Summary: Hearth T2: remote sensors not listed on the front panel

Firmware: 2.8.20260914dev-build512
When remote sensors are paired they are not shown as options on the front panel. Only the web dashboard lets the user switch to a remote sensor. If the user switches on the dashboard, the front panel does follow the selection.
Screenshot: attached by reporter.
```

Bug with a guess and a workaround:
```
Summary: Hearth app (Android): schedule change does not apply

Hearth app v3.1.4 (Android), Hearth T2 2.8.20260914dev-build512
Changing a setpoint in the weekly schedule sometimes does nothing, other times the thermostat restarts but the schedule stays the same. iOS app v3.2.0 works. Probably the app sends the new schedule before the thermostat finished the previous restart.

Steps:
1. Pair the thermostat with the app.
2. Open Hearth app, thermostat settings, Schedule.
3. Change Monday 06:00 from 19 to 21 degrees.

Actual: stays at 19, sometimes the thermostat restarts.
Expected: changes to 21, same as on iOS.
Workaround: change the setpoint from the front panel.
Video: attached by reporter.
```

Bug with a log line:
````
Summary: Hearth T2: history logging stops after switching to Away

2.9.20260918dev-build540
With history logging to the SD card on, I switched the thermostat from Home to Away while heating was running. Logging stopped, but the LOG icon on the display stayed lit. The file on the card ends at the moment of the switch. Heating continues. Reproduced 5 of 5 times, not on Hearth T1.

Log at the moment of the switch:
```
Sep 18 14:02:11 hearth daemon.err logger[812]: Writer: interval changed mid-file, closing output
```

Probably the logger closes the file when the interval changes and does not open a new one.
Workaround: turn logging off and on after switching modes.
````

Task:
```
Summary: Show last sync time on the thermostat details page

Currently the last sync time is only visible in the Info tab of the thermostat details page. Please move it next to the thermostat name, larger, with a clock icon.
Example: attached screenshot.
```

Task, one line:
```
Summary: Rename "Linked Devices" column to "Accessories"

Currently the "Linked Devices" column on the homes list shows "—" for every home, so users think nothing is linked. Please rename the column to "Accessories".
```

Comments:
```
Flipped the default sensor to Built-in on Hearth T2. Will be available on dev build #515.
```
```
2.8.20260916dev-build515
Factory reset now leaves the sensor on Built-in. Checked with Reset All and Reset Network.
Closing ticket.
```
```
2.8.20260916dev-build515
Still reproducible, follow the steps in the description. The sensor shows Remote after Reset All.
```
