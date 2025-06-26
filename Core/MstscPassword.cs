using System;
using System.ComponentModel;
using System.Runtime.InteropServices;
using System.Text;

namespace Core {
    /**
     * By ChipForster
     * https://www.remkoweijnen.nl/blog/2007/10/18/how-rdp-passwords-are-encrypted/
     */
    public class MstscPassword {
        private const int CryptProtectUiForbidden = 0x1;

        // Wrapper for the NULL handle or pointer.
        private static readonly IntPtr NullPtr = (IntPtr) 0;

        // Wrapper for DPAPI CryptProtectData function.
        [DllImport( "crypt32.dll", SetLastError = true, CharSet = CharSet.Auto )]

        private static extern bool CryptProtectData(
            ref DataBlob pPlainText,
            [MarshalAs(UnmanagedType.LPWStr)] string szDescription,
            IntPtr pEntroy,
            IntPtr pReserved,
            IntPtr pPrompt,
            int dwFlags,
            ref DataBlob pCipherText);

        // BLOB structure used to pass data to DPAPI functions.
        [StructLayout(LayoutKind.Sequential, CharSet = CharSet.Unicode)]
        private struct DataBlob {
            public int cbData;
            public IntPtr pbData;
        }

        private static void InitBlob(byte[] data, ref DataBlob blob) {
            blob.pbData = Marshal.AllocHGlobal(data.Length);
            if (blob.pbData == IntPtr.Zero)
                throw new Exception( "Unable to allocate buffer for BLOB data." );

            blob.cbData = data.Length;
            Marshal.Copy(data, 0, blob.pbData, data.Length);
        }

        public string EncryptPassword(string pw) {
            byte[] pwba = Encoding.Unicode.GetBytes(pw);
            DataBlob dataIn = new DataBlob();
            DataBlob dataOut = new DataBlob();
            StringBuilder epwsb = new StringBuilder();
            try {
                try {
                    InitBlob(pwba, ref dataIn);
                } catch (Exception ex) {
                    throw new Exception( "Cannot initialize dataIn BLOB.", ex );
                }

                bool success = CryptProtectData(
                    ref dataIn, 
                    "psw",
                NullPtr,
                NullPtr,
                NullPtr,
                CryptProtectUiForbidden,
                    ref dataOut );

                if (!success) {
                    int errCode = Marshal.GetLastWin32Error();
                    throw new Exception( "CryptProtectData failed.", new Win32Exception(errCode) );
                }

                byte[] epwba = new byte[dataOut.cbData];
                Marshal.Copy(dataOut.pbData, epwba, 0, dataOut.cbData);

                // Convert hex data to hex characters (suitable for a string)
                for (int i = 0; i < dataOut.cbData; i++) {
                    epwsb.Append(Convert.ToString(epwba[i], 16).PadLeft(2, '0').ToUpper());
                }

            } catch (Exception ex) {
                throw new Exception( "unable to encrypt data.", ex );
            } finally {
                if (dataIn.pbData != IntPtr.Zero)
                    Marshal.FreeHGlobal(dataIn.pbData);

                if (dataOut.pbData != IntPtr.Zero)
                    Marshal.FreeHGlobal(dataOut.pbData);
            }

            return epwsb.ToString();
        }
    }
}
