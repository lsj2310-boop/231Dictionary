package com.lsj231.interpreter;

import android.content.Context;
import android.content.SharedPreferences;
import android.security.keystore.KeyGenParameterSpec;
import android.security.keystore.KeyProperties;
import android.util.Base64;
import java.security.KeyStore;
import java.nio.charset.StandardCharsets;
import javax.crypto.Cipher;
import javax.crypto.KeyGenerator;
import javax.crypto.SecretKey;
import javax.crypto.spec.GCMParameterSpec;

public final class ApiKeyVault {
    private static final String ALIAS="231InterpreterGeminiKey";
    private final SharedPreferences prefs;
    public ApiKeyVault(Context context) {
        prefs=context.getApplicationContext().getSharedPreferences("gemini_key_encrypted",Context.MODE_PRIVATE);
    }
    private SecretKey key() throws Exception {
        KeyStore store=KeyStore.getInstance("AndroidKeyStore");
        store.load(null);
        KeyStore.Entry entry=store.getEntry(ALIAS,null);
        if(entry instanceof KeyStore.SecretKeyEntry) return ((KeyStore.SecretKeyEntry)entry).getSecretKey();
        KeyGenerator gen=KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES,"AndroidKeyStore");
        gen.init(new KeyGenParameterSpec.Builder(ALIAS,KeyProperties.PURPOSE_ENCRYPT|KeyProperties.PURPOSE_DECRYPT)
                .setBlockModes(KeyProperties.BLOCK_MODE_GCM)
                .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE).build());
        return gen.generateKey();
    }
    public synchronized boolean save(String secret) {
        if(secret==null || secret.trim().isEmpty())return false;
        try{
            Cipher cipher=Cipher.getInstance("AES/GCM/NoPadding");
            cipher.init(Cipher.ENCRYPT_MODE,key());
            String data=Base64.encodeToString(cipher.doFinal(secret.trim().getBytes(StandardCharsets.UTF_8)),Base64.NO_WRAP);
            String iv=Base64.encodeToString(cipher.getIV(),Base64.NO_WRAP);
            return prefs.edit().putString("data",data).putString("iv",iv).commit();
        }catch(Exception ignored){return false;}
    }
    public synchronized String read(){
        try{
            String data=prefs.getString("data",""),iv=prefs.getString("iv","");
            if(data.isEmpty()||iv.isEmpty())return "";
            Cipher cipher=Cipher.getInstance("AES/GCM/NoPadding");
            cipher.init(Cipher.DECRYPT_MODE,key(),new GCMParameterSpec(128,Base64.decode(iv,Base64.NO_WRAP)));
            return new String(cipher.doFinal(Base64.decode(data,Base64.NO_WRAP)),StandardCharsets.UTF_8);
        }catch(Exception ignored){return "";}
    }
    public boolean exists(){return !prefs.getString("data","").isEmpty();}
    public void clear(){prefs.edit().remove("data").remove("iv").apply();}
}
