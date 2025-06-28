using System;
using CommandLine;
using Core;
using Newtonsoft.Json;

namespace Cli {
    internal static class Program {
        internal class Options {
            // Optional config
            [Option('c', "config", Required = false, HelpText = "Path to the configuration folder.")]
            public string ConfigFolder { get; set; } = ".\\";
            
            // Optional json output flag
            [Option('j', "json", Required = false, HelpText = "Output in JSON format.")]
            public bool JsonOutput { get; set; } = false;
        }
        
        // Connect
        [Verb("connect", HelpText = "Connect to a profile")]
        class ConnectOptions: Options {
            // Name as verb too 
            [Value(0, MetaName = "name", Required = true, HelpText = "Name of the profile to connect to.")]
            public string Name { get; set; } = "";
        }
        
        // List
        [Verb("list", HelpText = "List all profiles")]
        class ListOptions: Options {
            
        }
        
        // Add
        [Verb("add", HelpText = "Add a new profile")]
        class AddOptions: Options {
            // Name
            [Option('n', "name", Required = true, HelpText = "Name of the profile.")]
            public string Name { get; set; } = "";
            
            // Computer
            [Option('c', "computer", Required = true, HelpText = "Computer name / Hostname / IP address (:port) (e.g. 192.168.1.2:3389)")]

            public string Computer { get; set; } = "";
            
            // Username
            [Option('u', "username", Required = true, HelpText = "Username")]

            public string Username { get; set; } = "";
            
            // Domain
            [Option('d', "domain", Required = false, HelpText = "Domain")]
            public string Domain { get; set; } = "";
            
            // Password
            [Option("password-unsafe", Required = false, HelpText = "Password (It will be recorded in terminal history, use with caution)")]
            public string Password { get; set; } = "";
        }
        
        // Edit
        [Verb("edit", HelpText = "Edit a profile")]
        class EditOptions : Options {
        }

        // Remove
        [Verb("remove", HelpText = "Remove a profile")]
        class RemoveOptions {
            
        }
        
        // Show the GUI to edit RDP options
        [Verb("show-more-options", HelpText = "Show the GUI to edit RDP options")]
        class EditRdpOptions : Options {
            // Name
            [Value(0, MetaName = "name", Required = true, HelpText = "Name of the profile to show more options for.")]
            public string Name { get; set; } = "";
        }
        

        public static int Main(string[] args) {
            try {
                return Parser.Default.ParseArguments<ConnectOptions, ListOptions, AddOptions, RemoveOptions>(args)
                    .MapResult(
                        (AddOptions options) => Add(options),
                        (ListOptions options) => List(options),
                        (ConnectOptions options) => Connect(options),
                        errs => 1
                    );
            } catch (Exception ex) {
                Console.WriteLine(ex.Message);
                return 1;
            }
        }

        public static Config GetConfig(Options options) {
            return Config.GetConfig(options.ConfigFolder);
        }
        
        private static int List(ListOptions options) {
            var config = GetConfig(options);

            if (options.JsonOutput) {
                Console.WriteLine(JsonConvert.SerializeObject(config.Profiles, Formatting.Indented));
                return 0;
            }
            
            if (config.Profiles.Count == 0) {
                Console.WriteLine("No profiles found.");
                return 0;
            }
            
            foreach (var profile in config.Profiles) {
                Console.WriteLine($"{profile.Name} - {profile.Computer}, {profile.Username}, {profile.Domain}");
            }
            return 0;
        }
        
        private static int Add(AddOptions options) {
            var config = GetConfig(options);
            
            var profile = new Profile {
                Name = options.Name,
                Computer = options.Computer,
                Username = options.Username,
                Domain = options.Domain,
                Config = config,
            };
            
            config.profileExists(profile);
            
            if (!string.IsNullOrEmpty(options.Password)) {
                profile.Password = options.Password;
            } else {
                Console.Write("Enter password: ");
                profile.Password = Util.ReadPassword();
            }
            
            profile.PrepareRdpFile();
            
            config.Profiles.Add(profile);
            config.Save();
            
            return 0;
        }
        
        private static int Connect(ConnectOptions options) {
            var config = GetConfig(options);
            
            foreach (var profile in config.Profiles) {
                if (profile.Name.Equals(options.Name, StringComparison.OrdinalIgnoreCase)) {
                    // Prepare the RDP file and open it
                    profile.PrepareRdpFile();
                    System.Diagnostics.Process.Start(profile.Filename);
                    return 0;
                }
            }
            
            Console.WriteLine($"Profile '{options.Name}' not found.");

            return 1;
        }
    }
}