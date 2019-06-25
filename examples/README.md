# Why Examples

This directory includes some small examples to illustrate the functionality of why. If you want to try these examples build the server or download the latest release, download the examples folder and create the following config.

### Config

```
{
  "PublicDir": "./examples",
  "EnableError": true,
  "BindAddress": ":8765",
  "Extensions": {
    "bbolt": ["store.bbolt", null],
  }
}
```

- Point the ``PublicDir`` to the downloaded ``./examples`` folder.
- Feel free to change the ``BindAddress`` to any ip:port you like.
- ``Extensions`` need to stay like that. The examples mostly use bbolt as storage so the extension needs to be loaded!
