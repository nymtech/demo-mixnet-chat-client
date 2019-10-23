import "semantic-ui-css/semantic.min.css"; // I don't even know... without it the LOCAL FILES would not properly get loaded
var exec = nodeRequire('child_process').exec

const fixPath = nodeRequire('fix-path');
fixPath();

// import "semantic-ui";
// const $: JQueryStatic = (window as any)["jQuery"];

const localhost: string = document.location.host || "localhost";
const port: string = "9000";
const fetchMsg = JSON.stringify({
	fetch: {},
});

const getRecipientsMsg = JSON.stringify({
	clients: {},
});


// doesn't have any fancy signatures, etc because I WON'T do elliptic crypto in js unless
// absolutely neccessary
interface ElectronChatMessage {
	content: string;
	senderPublicKey: string;
	senderProviderPublicKey: string;
}

interface ClientData {
	id: string;
	pubKey: string;
	provider: {
		id: string;
		host: string;
		port: string;
		pubKey: string;
	};
}

// 'chat2'
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

// OUR KEY (well, if I run it locally with '--id chat1')
const senderKey = "eqjn-P2hFQpowbVPfAwtN3wDVfSKAhDrjgQvGyoa10Y="


class SocketConnection {
	private conn: WebSocket;
	private ticker: number;
	private clients: ClientData[];
	constructor() {
		const conn = new WebSocket(`ws://${localhost}:${port}/mix`);
		conn.onclose = this.onSocketClose;
		conn.onmessage = this.onSocketMessage;
		conn.onerror = (ev: Event) => console.log("Socket error: ", ev);
		conn.onopen = this.getClients.bind(this);

		this.clients = [];
		this.conn = conn;
		this.ticker = window.setInterval(this.fetchMessages.bind(this), 1000);
	}

	public closeConnection() {
		this.conn.close();
		window.clearInterval(this.ticker);
	}

	public getClients() {
		console.log("getting list of available clients...");
		this.conn.send(getRecipientsMsg);
	}

	public sendMessage(message: string) {
		const selectedRecipientIdx = $("#recipientSelector").dropdown("get value");
		if (selectedRecipientIdx.length === 0) {
			return;
		}

		const selectedRecipient = this.clients[selectedRecipientIdx];

		// once recipient is selected, don't allow changing it
		if (!$("#recipientSelector").hasClass("disabled")) {
			$("#recipientSelector").addClass("disabled");
			// also update the sender divider here
			$("#senderDivider").html("Sending to " + this.formatDisplayedClient(selectedRecipient));
		}

		console.log(selectedRecipient);

		const sendMsg = JSON.stringify({
				send: {
					message: btoa(message),
					recipient: selectedRecipient,
				},
		});
		this.conn.send(sendMsg);
		createChatMessage("you", message, true);
	}

	private onSocketClose(ev: CloseEvent) {
		console.log("The websocket was closed", ev);

		const innerHeader = $("<div>", {
			class: "sub header",
			id: "wsclosedsub",
		}).text(`(code: ${ev.code}) - ${ev.reason || "no reason provided"}`);

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

	private handleFetchResponse(fetchData: any) {
		const messages = fetchData.fetch.messages;

		for (const msg of messages) {
			// TODO: js-chat message formatting + parsing
			const chatMessage: ElectronChatMessage = {
				content: "",
				senderProviderPublicKey: "bbbbbbbbbbbbbbbbbbbbb",
				senderPublicKey: "aaaaaaaaaaaaaaaaaaaaaaaa",
			};
			// TODO: later just do `const chatMessage = msg as ElectronChatMessage;`

			// TODO: do we need to decode it?
			chatMessage.content = atob(msg);
			createChatMessage(
				`??? - ${chatMessage.senderPublicKey.substring(0,8)}...@${chatMessage.senderProviderPublicKey.substring(0,8)}...`,
				chatMessage.content,
			);
		}
	}

	private formatDisplayedClient(client: ClientData): string {
		return "??? - " + client.id.substring(0, 8) + "...";
	}

	private handleClientsResponse(clientsData: any) {
		if (!$("#recipientSelector").hasClass("disabled")) {
			$("#recipientSelector").removeClass("disabled");
		}

		const availableClients = clientsData.clients.clients as ClientData[];
		// update our current list
		this.clients = availableClients;

		const valuesArray = availableClients.map((client, idx) => {
			return {name: this.formatDisplayedClient(client), value: idx};
		});

		$("#recipientSelector").dropdown({
			placeholder: "Choose recipient",
			fullTextSearch: true,
			values: valuesArray, // don't mind the errors, it's just typescript not really liking jquery
		});
	}

	// had to define it as an arrow function otherwise I couldn't call this.handle...
	private onSocketMessage = (ev: MessageEvent): void => {

		// we can either receive list of clients or actual message
		const receivedData = JSON.parse(ev.data);

		if (receivedData.hasOwnProperty("fetch")) {
			return this.handleFetchResponse(receivedData);
		} else if (receivedData.hasOwnProperty("clients")) {
			return this.handleClientsResponse(receivedData);
		} else if (receivedData.hasOwnProperty("send")) {
			console.log("received send confirmation");
		}

		console.log("Received unknown response!");
		console.log(receivedData);
	}

	private fetchMessages() {
		console.log("checking for new messages...");
		this.conn.send(fetchMsg);
	}



}

function createChatMessage(senderID: string, content: string, isReply: boolean = false) {
	const textDiv = $("<div>", {
		class: "text",
	}).text(content);

	// TODO: we really should get it from message itself rather than use local time, but oh well
	const dateDiv = $("<div>", {
		class: "date",
	}).text(new Date().toLocaleTimeString());

	const metadataDiv = $("<div>", {
		class: "metadata",
	}).append(dateDiv);

	const authorAnchor = $("<a>", {
		class: "author",
	}).text(senderID);

	const contentDiv = $("<div>", {
		class: "content",
	}).append(authorAnchor, metadataDiv, textDiv);

	const avatarAnchor = $("<a>", {
		class: "avatar",
	}).html('<img src="assets/avatar.png">');

	const commentDiv = $("<div>", {
		class: "comment",
	}).append(avatarAnchor, contentDiv);

	let chatMessageDiv: JQuery<HTMLElement>;
	if (isReply) {
		chatMessageDiv = $("<div>", {
			class: "chatMessage reply",
		});
	} else {
		chatMessageDiv = $("<div>", {
			class: "chatMessage incoming",
		});
	}
	chatMessageDiv.append(commentDiv);

	$("#messagesList").append(chatMessageDiv);
}

function handleSendAction(conn: SocketConnection) {
	const inputElement = $("#msgInput");
	const messageInput = inputElement.val() as string;
	if (messageInput.length === 0) {
		return; // don't do anything if there's nothing to send
	}

	conn.sendMessage(messageInput);

	// finally clear input box
	inputElement.val("");
}

function main() {
	const conn = new SocketConnection();

	$("#closeWS").click(() => {
		conn.closeConnection();
	});
	$("#sendMsg").click(() => {
		conn.sendMessage("foomp");
	});
	$("#getClients").click(() => {
		conn.getClients();
	});
	$("#sendBtn").click(() => {
		handleSendAction(conn);
	});
	$("#msgInput").on("keydown", (ev: JQuery.KeyDownEvent) => {
		if (ev.keyCode === 13) {
			handleSendAction(conn);
		}
	});
}

$(document).ready(() => main());


