// The module 'vscode' contains the VS Code extensibility API
// Import the module and reference it with the alias vscode in your code below
import { ExtensionContext, languages, commands, Disposable, workspace} from 'vscode';
import { CodelensProvider } from './CodelensProvider';
import fetch from 'node-fetch';
import * as vscode from 'vscode';

// this method is called when your extension is activated
// your extension is activated the very first time the command is executed

let disposables: Disposable[] = [];

export function activate(context: ExtensionContext) {
	const codelensProvider = new CodelensProvider();

	languages.registerCodeLensProvider("*", codelensProvider);

	commands.registerCommand("modernizer-vscode.enableCodeLens", () => {
		workspace.getConfiguration("modernizer-vscode").update("enableCodeLens", true, true);
	});

	commands.registerCommand("modernizer-vscode.disableCodeLens", () => {
		workspace.getConfiguration("modernizer-vscode").update("enableCodeLens", false, true);
	});

	commands.registerCommand("modernizer-vscode.codelensAction", async (args: any) => {
		try {
			const activeEditor = vscode.window.activeTextEditor;
			if (!activeEditor) {
				vscode.window.showWarningMessage('No active text editor found.');
				return;
			}
	
			// Get the selected function range
			const selectedFunctionRange = getSelectedFunctionRange(activeEditor);
			if (!selectedFunctionRange) {
				vscode.window.showWarningMessage('No function selected. Please select a function to generate a prompt.');
				return;
			}
	
			// Extract the function code from the range
			const functionCode = activeEditor.document.getText(selectedFunctionRange);
	
			// Send the function code as a prompt to the Ollama API
			const response = await fetch('http://192.168.10.163:8080/generate', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({
					model: 'codellama:13b-instruct',
					prompt: `${functionCode}`,
					stream: false
				})
			});
	
			if (response.ok) {
				try {
					const contentType = response.headers.get('content-type');
					if (contentType && contentType.includes('application/json')) {
						// Parse and display the Ollama response
						const responseBody = await response.json();
						const responseText = responseBody || 'No response field found';
	
						const outputWindow = vscode.window.createOutputChannel('Ollama Response');
						outputWindow.show(true);
						outputWindow.append(responseText);
	
						vscode.window.showInformationMessage(responseText);
					} else {
						vscode.window.showWarningMessage('Received non-JSON response. Check the API for possible errors.');
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
	});

	commands.registerCommand("modernizer-vscode.retrieveResponse", async (args: any) => {
		try {
			const activeEditor = vscode.window.activeTextEditor;
			if (!activeEditor) {
				vscode.window.showWarningMessage('No active text editor found.');
				return;
			}
	
			// Get the selected function range
			const selectedFunctionRange = getSelectedFunctionRange(activeEditor);
			if (!selectedFunctionRange) {
				vscode.window.showWarningMessage('No function selected. Please select a function to generate a prompt.');
				return;
			}
	
			// Extract the function code from the range
			const functionCode = activeEditor.document.getText(selectedFunctionRange);
	
			// Send the function code as a prompt to the Ollama API
			const response = await fetch('http://192.168.10.163:8080/generate', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({
					model: 'codellama:13b-instruct',
					prompt: `${functionCode}`,
					stream: false
				})
			});
	
			if (response.ok) {
				try {
					const contentType = response.headers.get('content-type');
					if (contentType && contentType.includes('application/json')) {
						// Parse and display the Ollama response
						const responseBody = await response.json();
						const responseText = responseBody || 'No response field found';
	
						const outputWindow = vscode.window.createOutputChannel('Ollama Response');
						outputWindow.show(true);
						outputWindow.append(responseText);
	
						vscode.window.showInformationMessage(responseText);
					} else {
						vscode.window.showWarningMessage('Received non-JSON response. Check the API for possible errors.');
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
	});
	
}


// this method is called when your extension is deactivated
export function deactivate() {
	if (disposables) {
		disposables.forEach(item => item.dispose());
	}
	disposables = [];
}

export function getWebviewContent(responseText: string): string {
    // Use a basic HTML template to display the responseText
    return `
        <!DOCTYPE html>
        <html lang="en">
        <head>
            <meta charset="UTF-8">
            <meta name="viewport" content="width=device-width, initial-scale=1.0">
            <title>Ollama Response</title>
        </head>
        <body>
            <pre>${responseText}</pre>
        </body>
        </html>
    `;
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

