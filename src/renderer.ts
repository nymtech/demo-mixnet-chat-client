import "semantic-ui-css/semantic.min.css"; // I don't even know... without it the LOCAL FILES would not properly get loaded

// import "semantic-ui";
// const $: JQueryStatic = (window as any)["jQuery"];

const localhost: string = document.location.host || "localhost";
const port: string = "9000";
const fetchMsg = JSON.stringify({
	fetch: {},
});

// function attachHandlers() {
//   let msg = document.getElementById("msg");
//   let log = document.getElementById("log");
//   let form = document.getElementById("form");



// document.getElementById("form").onsubmit = () => {
// 	if (!conn) {
// 		return false;
// 	}
// 	if (!msg.value) {
// 		return false;
// 	}
// 	conn.send(msg.value);
// 	msg.value = "";
// 	return false;
// };


// document.getElementById("fetchBtn").onclick = () => {
// 	let recipientSelector = document.getElementById("recipientSelect");
// 	let chosenRecipient = recipientSelector.options[recipientSelector.selectedIndex].value;


// 	let fetchMsg = JSON.stringify({
// 		"fetch": {}
// 	});
// 	conn.send(fetchMsg)
// };


// function appendLog(item) {
// 	let doScroll = log.scrollTop > log.scrollHeight - log.clientHeight - 1;
// 	log.appendChild(item);
// 	if (doScroll) {
// 		log.scrollTop = log.scrollHeight - log.clientHeight;
// 	}
// }


const hardcodedRecipient = {
		"recipient": {
		"id": "1I1XFLNq9fIP7gDcmJZNH6GtCk5r9-wb3Ay_fZa9fnI=",
		"pubKey": "1I1XFLNq9fIP7gDcmJZNH6GtCk5r9+wb3Ay/fZa9fnI=",
		"provider": {
			"id": "XiVE6xA10xFkAwfIQuBDc_JRXWerL0Pcqi7DipEUeTE=",
			"host": "3.8.176.11",
			"port": "1789",
			"pubKey": "XiVE6xA10xFkAwfIQuBDc/JRXWerL0Pcqi7DipEUeTE="
		}
	}
}

const senderKey = "eqjn-P2hFQpowbVPfAwtN3wDVfSKAhDrjgQvGyoa10Y="


class SocketConnection {
	private conn: WebSocket;
	private ticker: number;
	constructor() {
		const conn = new WebSocket(`ws://${localhost}:${port}/mix`);
		conn.onclose = this.onSocketClose;
		conn.onmessage = this.onSocketMessage;
		conn.onerror = (ev: Event) => console.log("Socket error: ", ev);
		conn.onopen = () => console.log("Socket was opened!");

		this.conn = conn;

		this.ticker = window.setInterval(this.fetchMessages.bind(this), 1000);
	}

	public closeConnection() {
		this.conn.close();
		window.clearInterval(this.ticker);
	}

	public sendMessage(message: string, recipient = hardcodedRecipient.recipient) {
		const sendMsg = JSON.stringify({
				send: {
					message: btoa(message),
					recipient,
				},
		});
		this.conn.send(sendMsg);
	}

	private onSocketClose(ev: CloseEvent) {
		console.log("The websocket was closed", ev);

		const innerHeader = $("<div>", {
			class: "sub header",
			id: "wsclosedsub",
		}).text(`(code: ${ev.code}) - ${ev.reason || "unknown"}`);

		const contentDiv = $("<div>", {
			class: "content",
			id: "wsclosedcontent",
		}).text("The websocket was closed.").append(innerHeader);

		const closedIcon = $("<i>", {
			class: "close icon",
			id: "wsclosedicon",
		});

		const closedHeader = $("<h2>", {
			class: "ui icon header",
			id: "wsclosedheader",
		}).append(closedIcon, contentDiv);

		$("#noticeDiv").append(closedHeader);
	}

	private onSocketMessage(ev: MessageEvent) {

		// conn.onmessage = (evt: MessageEvent) => {
		// 	let res = JSON.parse(evt.data)

		// 	// THIS IS ONLY FOR FETCH
		// 	for (msg in res["messages"]) {
		// 		let item = document.createElement("div");
		// 		item.innerText = "Received: " + msg;
		// 		appendLog(item)
		// 	}
		// 	};

		console.log("received message", ev.data);
	}

	private fetchMessages() {
		console.log("checking for new messages...");
		this.conn.send(fetchMsg);
	}


}



function main() {
	const conn = new SocketConnection();

	// t = document.getElementById("connectWS")
	// t.onclick(() => console.log("clicked"))
	// $.ge
	$("#closeWS").click(() => {
		conn.closeConnection()
	});

	$("#sendMsg").click(() => {


		conn.sendMessage("foomp");

	});


	// makeSocketConnection()
	// document.getElementById("test").innerText = "foo"
	// makeSocketConnection()
}

main();
