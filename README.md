# tago

*Simple file tagging utility.*

You can store additional information about files using *.tago files.

*.tago files look like this.

```
some key : value

other key : [
    Some long value spanning
    multiple lines.
]
```

For example, let's say you did `tago poem.txt`.

It first look for **poem.tago**. Then it looks for **tago.tago** in the same directory.

Then it goes up through directories to find other **tago.tago** files.

Then it prints information stored in *.tago files.

# License

This project is under MIT License.