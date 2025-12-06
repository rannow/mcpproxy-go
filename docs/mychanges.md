1. check Menu to "stop all Servers" the Tool by stoping/killing alle processes
2. check that all connected Server can connet all Tools and the Tools are working correct => do a test of tools
3. Test the Diagnostic Tool to Fix the Problem of starting Servers, Check that the Diagnostic Tool can run over all Servers that not Starting Correct
4. Check the Funktion to Start the Server with lasys Loding
5. Check that the Reload of the Config Checks for changes and Restart the Servers
6. check that disabled 

tray changes
1. Sort menus by name

can you use claude flow to analyse the start process and checke that all Server get connectec in batches and if thy can not get connected correct after 5 times thy get shown as automatic disabled and a report is created with the error case and detailed infos why the server could not start, like packges missing, timeouts, errors, run the Server with a config that has all MCP-Server marked as active, check what happend with the info stored in the DB.  Please create a detailde documentation and check that no left overers are avalibal that can consume stille resources if the server is not started and check in details that all server starte correct without consuming mroe than needed resources and can shut down gracefuly or if not possible shut down by killing ths process after timeout

--------------------------------
Can you add a Web Page that can save Secrets in a key store that is encrypted and can be used by the application to access the secrets without having to store them in the Config or the Environment Variables
can you extend the config loader to support the stored secrets in the key store and update the Readme.md file to reflect the new feature

 Can you use claude Flow to make a detaild analyse of the Code for refactoring, check for old and inconsistent code changees, make a detailed analysis about the state and a consistent update with all state types also the Auto-Disabled and the Start Process and stop process, that servers that not worke are killed, that if a server is not shuting down gracfuly he is also killed aftr time x, this is for a single Server Restart or if all Server are Stoped, check the Procress to stop all Server and to quit the full Application
-----
ich möchte gerne einen Agent erstellen der mir hilft problem von mcp-servern zu                                analysieren er soll dabei 1. eine Detaild analyse erstellen mit möglichen                                                                       
problemen und eine Liste mit fragen wie z.b. für misssing environment daten                                                                     
oder hinweisen für die fehler behebung. Das ganze soll als MD Datei gespeichert                                                                 │
werden. Dabei soll er fehler von mcpproxy upsteram Server analysieren, dazu                                                                     │
kann er die mcp_config.jaon lesen und die links darin über die Server                                                                           │
faild_servers.log in /Users/hrannow/.mcpproxy/ und die logs in                                                                                  │
/Users/hrannow/Library/Logs/mcpproxy to direct test you may can use                                                                             │
 https://github.com/wong2/mcp-cli            