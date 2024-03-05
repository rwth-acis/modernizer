// The module 'vscode' contains the VS Code extensibility API
// Import the module and reference it with the alias vscode in your code below
import { ExtensionContext, languages, commands, Disposable, workspace} from 'vscode';
import { CodelensProvider } from './CodelensProvider';
import fetch from 'node-fetch';
import * as vscode from 'vscode';
import { DisplayVoting } from './VotingMechanism';
import { calculateURL } from './Util';

// this method is called when your extension is activated
// your extension is activated the very first time the command is executed

let disposables: Disposable[] = [];


export function activate(context: ExtensionContext) {

	const codelensProvider = new CodelensProvider();

	context.subscriptions.push(disposableUserInput);
    context.subscriptions.push(disposableGetGitURL);

	languages.registerCodeLensProvider("*", codelensProvider);

	commands.registerCommand("modernizer-vscode.enableCodeLens", () => {
		workspace.getConfiguration("modernizer-vscode").update("enableCodeLens", true, true);
	});

	commands.registerCommand("modernizer-vscode.disableCodeLens", () => {
		workspace.getConfiguration("modernizer-vscode").update("enableCodeLens", false, true);
	});

	context.subscriptions.push(disposableCustomPrompt);
	context.subscriptions.push(disposableRandomPrompt);
}


// this method is called when your extension is deactivated
export function deactivate() {
	if (disposables) {
		disposables.forEach(item => item.dispose());
	}
	disposables = [];
}

// Helper function to get the selected function range
export function getSelectedFunctionRange(editor: vscode.TextEditor): vscode.Range | undefined {
    const selection = editor.selection;
    const document = editor.document;

    if (!selection.isEmpty) {
        // If there is a selection, use the selected range
        return new vscode.Range(selection.start, selection.end);
    } else {
        // If no selection, find the current function
        const cursorPosition = selection.start;
        let line = document.lineAt(cursorPosition);

        // Find the start of the function
        let startLine = cursorPosition.line;
        while (startLine > 0 && !line.text.trim().startsWith('func')) {
            startLine--;
            line = document.lineAt(startLine);
        }

        // Find the end of the function
        let endLine = cursorPosition.line;
        while (endLine < document.lineCount - 1 && !line.text.trim().endsWith('}')) {
            endLine++;
            line = document.lineAt(endLine);
        }

        // Return the range of the found function
        return new vscode.Range(startLine, 0, endLine, line.text.length);
    }
}

let disposableUserInput = vscode.commands.registerCommand('modernizer-vscode.customPrompt2', async () => {
	const editor = vscode.window.activeTextEditor;
	if (!editor) {
		vscode.window.showErrorMessage("No text selected.");
		return;
	}

	const userInput = await vscode.window.showInputBox({
		prompt: "Enter instruct to use in prompt"
	});

	if (!userInput) {
		vscode.window.showErrorMessage("No input provided.");
		return;
	}

	vscode.window.showInformationMessage(`Notification: ${userInput}`);
});

let disposableRandomPrompt = vscode.commands.registerCommand('modernizer-vscode.randomPrompt', async () => {
	try {
		await generateRandomPrompt();
	} catch (error: any) {
		vscode.window.showErrorMessage(`Error: ${error.message}`);
	}
});

let disposableCustomPrompt = vscode.commands.registerCommand('modernizer-vscode.customPrompt', async () => {
	try {
		await generateCustomPrompt();
	} catch (error: any) {
		vscode.window.showErrorMessage(`Error: ${error.message}`);
	}
});

let disposableGetGitURL = vscode.commands.registerCommand('modernizer-vscode.gitURL', async () => {
	try {
		let URL = await calculateURL();
        vscode.window.showInformationMessage(`Notification: ${URL}`);
	} catch (error: any) {
		vscode.window.showErrorMessage(`Error: ${error.message}`);
	}
});

async function generateRandomPrompt() {
    const activeEditor = vscode.window.activeTextEditor;
    if (!activeEditor) {
        vscode.window.showWarningMessage('No active text editor found.');
        return;
    }

    const gitURL = await calculateURL();

    // Get the selected function range
    const selectedFunctionRange = getSelectedFunctionRange(activeEditor);
    if (!selectedFunctionRange) {
        vscode.window.showWarningMessage('No function selected. Please select a function to generate a prompt.');
        return;
    }

    // Extract the function code from the range
    const functionCode = activeEditor.document.getText(selectedFunctionRange);

    const baseUrl: string = vscode.workspace.getConfiguration("modernizer-vscode").get("baseURL", "https://modernizer.milki-psy.dbis.rwth-aachen.de");
    const generateRoute: string = '/generate';
    const url: string = `${baseUrl}${generateRoute}`;

    const promptData = {
        model: 'codellama:13b-instruct',
        prompt: `${functionCode}`,
        gitURL: gitURL,
    };

    await sendPromptToAPI(url, promptData);
}

async function generateCustomPrompt() {
    const activeEditor = vscode.window.activeTextEditor;
    if (!activeEditor) {
        vscode.window.showErrorMessage("No text selected.");
        return;
    }

    const gitURL = await calculateURL();

    const selectedFunctionRange = getSelectedFunctionRange(activeEditor);
    if (!selectedFunctionRange) {
        vscode.window.showWarningMessage('No function selected. Please select a function to generate a prompt.');
        return;
    }

    const functionCode = activeEditor.document.getText(selectedFunctionRange);

    const userInput = await vscode.window.showInputBox({
        prompt: "Enter instruct to use in prompt"
    });

    if (!userInput) return; // User canceled input

    const baseUrl: string = vscode.workspace.getConfiguration("modernizer-vscode").get("baseURL", "https://modernizer.milki-psy.dbis.rwth-aachen.de");
    const generateRoute: string = "/generate";
    const url: string = `${baseUrl}${generateRoute}`;

    const promptData = {
        model: "codellama:13b-instruct",
        prompt: `${functionCode}`,
        instruct: userInput,
        gitURL: gitURL,
    };

    await sendPromptToAPI(url, promptData);
}

async function sendPromptToAPI(url: string, promptData: any) {
    try {
        const response = await fetch(url, {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify(promptData)
        });

        if (response.ok) {
            try {
                const contentType = response.headers.get("content-type");
                if (contentType && contentType.includes("application/json")) {
                    // Parse and display the Ollama response
                    const responseBody = await response.json();
                    const responseText = responseBody.response || "No response field found";

                    const outputWindow = vscode.window.createOutputChannel("Ollama Response");
                    outputWindow.show(true);

                    outputWindow.append(`Generated new response with the instruct: ${responseBody.instruct}\n\n`);
                    outputWindow.append(responseText + "\n");

                    if (responseBody.promptID) {
                        DisplayVoting(responseBody.promptID);
                    } else {
                        vscode.window.showWarningMessage("No promptId field found");
                    }
                } else {
                    vscode.window.showWarningMessage("Received non-JSON response. Check the API for possible errors.");
                }
            } catch (jsonError: any) {
                vscode.window.showErrorMessage(`Failed to parse JSON response: ${jsonError.message}`);
            }
        } else {
            vscode.window.showErrorMessage(`Failed to make request: ${response.statusText}`);
        }
    } catch (error: any) {
        vscode.window.showErrorMessage(`Error: ${error.message}`);
    }
}