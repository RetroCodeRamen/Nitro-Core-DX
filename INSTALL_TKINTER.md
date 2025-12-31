# Installing tkinter for File Dialogs

The file dialogs (Open ROM and Dump) require tkinter to be installed.

## Linux (Ubuntu/Debian)
```bash
sudo apt-get install python3-tk
```

## macOS
tkinter should come with Python. If not:
```bash
brew install python-tk
```

## Windows
tkinter should come with Python by default.

## Verify Installation
After installing, verify with:
```bash
python3 -c "import tkinter; print('tkinter available')"
```

## Fallback Behavior
If tkinter is not available:
- **Dump**: Saves to current directory with default filename
- **Open ROM**: Shows error message (you can still load ROMs via command line)
