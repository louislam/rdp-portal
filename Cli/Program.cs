using System;
using CommandLine;

namespace Cli {
    internal class Program {
        private class Options {
            
            // Optional config
            [Option('c', "config", Required = false, HelpText = "Path to the configuration file.")]
            public string ConfigPath { get; set; } = "config.json";
    
        }
        
        // Connect
        [Verb("connect", HelpText = "Connect to a profile")]
        class ConnectOptions {

        }
        
        // List
        [Verb("list", HelpText = "List all profiles")]
        class ListOptions {
            
        }
        
        // Add
        [Verb("add", HelpText = "Add a new profile")]
        class AddOptions: Options {
            // Username
            [Option('u', "username", Required = true, HelpText = "Username for the profile.")]
            public string Username { get; set; }
        }
        
        // Remove
        [Verb("remove", HelpText = "Remove a profile")]
        class RemoveOptions {
            
        }
        

        public static int Main(string[] args) {
            return Parser.Default.ParseArguments<ConnectOptions, ListOptions, AddOptions, RemoveOptions>(args)
                .MapResult(
                    (AddOptions options) => Add(options),
                    errs => 1
                );
        }
        
        private static int Add(AddOptions options) {
            Console.WriteLine(options.ConfigPath);
            Console.WriteLine(options.Username);
            return 0;
        }
    }
}