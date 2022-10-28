using System;
using System.Linq;
using System.Security.Cryptography;
using System.Text;

public static class StringUtil {
    private static byte[] key = new byte[8] {226, 175, 129, 22, 187, 4, 143, 171};
    private static byte[] iv = new byte[8] {181, 239, 77, 174, 238, 148, 206, 164};

    public static string Encrypt(this string text) {
        SymmetricAlgorithm algorithm = DES.Create();
        ICryptoTransform transform = algorithm.CreateEncryptor(key, iv);
        byte[] inputbuffer = Encoding.Unicode.GetBytes(text);
        byte[] outputBuffer = transform.TransformFinalBlock(inputbuffer, 0, inputbuffer.Length);
        return Convert.ToBase64String(outputBuffer);
    }

    public static string Decrypt(this string text) {
        SymmetricAlgorithm algorithm = DES.Create();
        ICryptoTransform transform = algorithm.CreateDecryptor(key, iv);
        byte[] inputbuffer = Convert.FromBase64String(text);
        byte[] outputBuffer = transform.TransformFinalBlock(inputbuffer, 0, inputbuffer.Length);
        return Encoding.Unicode.GetString(outputBuffer);
    }

    public static string GenerateName(int len) {
        Random random = new Random();
        const string chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";
        return new string(Enumerable.Repeat(chars, len)
            .Select(s => s[random.Next(s.Length)]).ToArray());
    }
}
