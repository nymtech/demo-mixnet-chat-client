import { app, BrowserWindow, ipcMain } from "electron";
const path = require("path");
const execFile = require("child_process").execFile
const fixPath = require("fix-path");
const getPort = require("get-port");
fixPath();

let mainWindow: Electron.BrowserWindow;

async function onReady() {
	if (app.isPackaged) {
		process.argv.unshift(""); // temp workaround
	}
	if (process.argv.length < 3) {
		throw new Error("Insufficient number of arguments provided");
	}
	const loopixID = process.argv[2];
	let loopixPort: string;
	if (process.argv.length > 3) {
		loopixPort = process.argv[3];
	} else {
		loopixPort = await getPort();
	}

	console.log(`Chosen port: ${loopixPort}`);
	
	const loopixClient = execFile(path.resolve("dist/loopix-client"),
		["socket", "--id", loopixID, "--socket", "websocket", "--port", loopixPort],
		(error: any, stdout: any, stderr: any) => {
		if (error) {
			console.error("stderr", stderr);
			throw error;
		}
		console.log("stdout", stdout);
	});

	loopixClient.on("exit", (code: any) => {
		throw new Error(`Exit with code: ${code}`);
	});

	// listen for port requests from window we're about to spawn
	ipcMain.once("port", (event) => {
		event.returnValue = loopixPort;
	});

	// Create the browser window.
	mainWindow = new BrowserWindow({
		height: 1000,
		webPreferences: {
			nodeIntegration: true,
		  },
		width: 800,
	});


	// and load the index.html of the app.
	mainWindow.loadFile(path.resolve("dist/index.html"));

	// Open the DevTools.
	if (!app.isPackaged) {
		mainWindow.webContents.openDevTools();
	}

	mainWindow.on("close", () => {
		loopixClient.kill("SIGINT");
		app.quit();
	});

	app.on("quit", () => {
		loopixClient.kill("SIGINT");
	});
}

// This method will be called when Electron has finished
// initialization and is ready to create browser windows.
// Some APIs can only be used after this event occurs.
app.on("ready", onReady);

// Quit when all windows are closed.
app.on("window-all-closed", () => {
  // On OS X it is common for applications and their menu bar
  // to stay active until the user quits explicitly with Cmd + Q
  if (process.platform !== "darwin") {
	app.quit();
  }
});

app.on("activate", () => {
  // On OS X it"s common to re-create a window in the app when the
  // dock icon is clicked and there are no other windows open.
  if (mainWindow === null) {
	onReady();
  }
});

// In this file you can include the rest of your app"s specific main process
// code. You can also put them in separate files and require them here.