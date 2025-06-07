#include <iostream>
#include <cstdlib>
#include <filesystem>
#include <fstream>
#include <vector>

// Yardım menüsü
void printHelp() 
{
    std::cout << "anitr-cli kullanımı:\n"
              << "  --help, -h: Bu yardım menüsünü gösterir\n"
              << "  --gen-config: rofi-flags.conf dosyasını oluşturur\n"
              << "\n";
}

// rofi.flags.conf dosyasını oluşturacak funksiyon
void generateConfigFile() 
{
    std::string configDir = std::string(getenv("HOME")) + "/.config/anitr-cli";
    std::string configFile = configDir + "/rofi-flags.conf";

    // Klasörü oluştur (varsa atla)
    std::filesystem::create_directories(configDir);

    // Konfigürasyon dosyası varsa, uyarı göster ve çık
    if (std::filesystem::exists(configFile)) 
    {
        std::cout << "Konfigürasyon dosyası zaten var: " << configFile << "\n";
        return;
    }

    // Varsayılan parametrelerle dosya oluştur
    std::ofstream config(configFile);
    if (config.is_open()) 
    {
        config << "";
        config.close();
        std::cout << "Konfigürasyon dosyası oluşturuldu: " << configFile << "\n";
    } 
    
    else 
    {
        std::cerr << "Konfigürasyon dosyası oluşturulamıyor: " << configFile << "\n";
    }
}

std::string getSelectedSourceWithRofi(const std::vector<std::string>& options) 
{
    // Rofi parametrelerini okumak için konfigürasyon dosyasını kontrol et
    std::string rofi_flags = "";
    std::ifstream config_file(std::string(getenv("HOME")) + "/.config/anitr-cli/rofi-flags.conf");

    if (config_file.is_open()) 
    {
        std::string line;
        while (std::getline(config_file, line)) 
        {
            if (!line.empty() && line[0] != '#') 
            {
                rofi_flags += line + " ";
            }
        }
        config_file.close();
    }

    // Seçenekleri Rofi ile göster
    std::string rofi_cmd = "echo -e \"";

    for (size_t i = 0; i < options.size(); ++i) 
    {
        rofi_cmd += options[i];
        if (i != options.size() - 1)
        {
            rofi_cmd += "\\n";
        }
    }

    rofi_cmd += "\" | rofi -dmenu -p 'Kaynak seçin' " + rofi_flags + " 2>/dev/null";

    std::string selected;
    FILE* fp = popen(rofi_cmd.c_str(), "r");
    if (fp != NULL) 
    {
        char buffer[1024];
        if (fgets(buffer, sizeof(buffer), fp) != NULL) 
        {
            selected = buffer;
        }
        fclose(fp);
    }

    if (!selected.empty() && selected.back() == '\n') 
    {
        selected.pop_back();
    }

    return selected;
}


int main(int argc, char* argv[]) {
    // Komut satırı parametrelerini kontrol et
    for (int i = 1; i < argc; ++i) 
    {
        std::string arg = argv[i];

        if (arg == "--help" || arg == "-h") 
        {
            printHelp();
            return 0;
        } 
        
        else if (arg == "--gen-config") 
        {
            generateConfigFile();
            return 0;
        }
    }

    std::vector<std::string> source_options = {
	    "AnimeciX", "OpenAnime", "Çık"
    };

    std::string selectedSource = getSelectedSourceWithRofi(source_options);

    if (selectedSource == "AnimeciX") {
        system("animecix-cli");
    } else if (selectedSource == "OpenAnime") {
        system("openanime-cli");
    } else {
        exit(1);
    }

    return 0;
}


