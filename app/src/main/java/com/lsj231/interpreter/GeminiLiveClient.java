package com.lsj231.interpreter;

import android.media.AudioFormat;
import android.media.AudioRecord;
import android.media.MediaRecorder;
import android.os.Handler;
import android.os.Looper;
import android.util.Base64;
import org.json.JSONArray;
import org.json.JSONObject;
import okhttp3.OkHttpClient;
import okhttp3.Request;
import okhttp3.Response;
import okhttp3.WebSocket;
import okhttp3.WebSocketListener;
import java.util.concurrent.TimeUnit;

public final class GeminiLiveClient {
    public interface Listener {
        void status(String text);
        void interim(String text);
        void committed(String text);
        void failure(String text);
    }
    private static final String ENDPOINT="wss://generativelanguage.googleapis.com/ws/google.ai.generativelanguage.v1beta.GenerativeService.BidiGenerateContent";
    private final Handler main=new Handler(Looper.getMainLooper());
    private final Listener listener;
    private final OkHttpClient client=new OkHttpClient.Builder()
        .connectTimeout(15,TimeUnit.SECONDS).pingInterval(20,TimeUnit.SECONDS)
        .readTimeout(0,TimeUnit.MILLISECONDS).build();
    private volatile boolean active=false,ready=false;
    private volatile WebSocket socket;
    private volatile AudioRecord recorder;
    public GeminiLiveClient(Listener l){listener=l;}
    private void ui(Runnable r){main.post(r);}
    public void connect(String key,String lang) {
        if(key==null || key.isEmpty()){ui(()->listener.failure("Gemini API 키가 없습니다."));return;}
        active=true;
        Request request=new Request.Builder().url(ENDPOINT+"?key="+key).build();
        socket=client.newWebSocket(request,new WebSocketListener(){
            @Override public void onOpen(WebSocket ws,Response r){
                if(!active){ws.cancel();return;}
                try{
                    JSONObject transcribe=new JSONObject()
                        .put("languageCodes",new JSONArray().put(lang))
                        .put("mode","SMART");
                    if(lang.equals("en-US")||lang.equals("ko-KR"))
                        transcribe.put("customVocabulary",new JSONArray()
                            .put("Tata Daewoo Mobility").put("MAXEN").put("KUXEN")
                            .put("GIXEN").put("NOVUS").put("FPT").put("WABCO")
                            .put("warranty").put("quality assurance").put("PDI")
                            .put("Cursor 9").put("CSMS"));
                    JSONObject setup=new JSONObject().put("setup",new JSONObject()
                        .put("model","models/gemini-3.5-transcribe-live")
                        .put("generationConfig",new JSONObject()
                            .put("responseModalities",new JSONArray().put("TEXT")))
                        .put("inputAudioTranscription",transcribe));
                    ws.send(setup.toString());
                    ui(()->{if(active)listener.status("Gemini 연결됨 · 준비 중");});
                }catch(Exception e){
                    ui(()->{if(active)listener.failure("Gemini 세션 설정에 실패했습니다.");});
                }
            }
            @Override public void onMessage(WebSocket ws,String message){
                if(!active)return;
                try{
                    JSONObject json=new JSONObject(message);
                    if(json.has("setupComplete")&&!ready){
                        ready=true;ui(()->{if(active)listener.status("● Gemini 실시간 음성 인식 중");});
                        startMic();
                    }
                    JSONObject contents=json.optJSONObject("serverContent");
                    if(contents==null)return;
                    JSONObject interim=contents.optJSONObject("interimInputTranscription");
                    if(interim!=null){
                        String text=interim.optString("text","").trim();
                        if(!text.isEmpty())ui(()->{if(active)listener.interim(text);});
                    }
                    JSONObject finalText=contents.optJSONObject("inputTranscription");
                    if(finalText!=null){
                        String text=finalText.optString("text","").trim();
                        if(!text.isEmpty())ui(()->{if(active)listener.committed(text);});
                    }
                }catch(Exception ignored){}
            }
            @Override public void onFailure(WebSocket ws,Throwable error,Response r){
                if(!active)return;
                int status=r==null?0:r.code();
                String msg=status==401||status==403?"Gemini API 인증 실패 (HTTP "+status+"). 키와 권한을 확인하세요."
                   :status==429?"Gemini API 사용량 한도를 초과했습니다."
                   :"Gemini 연결이 끊겼습니다"+(status>0?" (HTTP "+status+")":"")+". 인터넷·모델 사용 가능 여부를 확인하세요.";
                ui(()->{if(active)listener.failure(msg);});
            }
            @Override public void onClosed(WebSocket ws,int status,String reason){
                ui(()->{if(active)listener.failure("Gemini 연결이 종료되었습니다 (코드 "+status+").");});
            }
        });
    }
    private void startMic(){
        new Thread(()->{
            int sampleRate=16000;
            int minimum=AudioRecord.getMinBufferSize(sampleRate,AudioFormat.CHANNEL_IN_MONO,AudioFormat.ENCODING_PCM_16BIT);
            if(minimum<=0){ui(()->{if(active)listener.failure("마이크가 16 kHz 녹음 형식을 지원하지 않습니다.");});return;}
            AudioRecord audio=null;
            try{
                audio=new AudioRecord(MediaRecorder.AudioSource.VOICE_RECOGNITION,sampleRate,
                    AudioFormat.CHANNEL_IN_MONO,AudioFormat.ENCODING_PCM_16BIT,Math.max(minimum*2,16000));
                if(audio.getState()!=AudioRecord.STATE_INITIALIZED){
                    audio.release();ui(()->{if(active)listener.failure("마이크를 시작할 수 없습니다.");});return;
                }
                recorder=audio;
                audio.startRecording();
                byte[] buffer=new byte[3200]; // 100ms of 16-bit PCM at 16kHz
                while(active&&ready){
                    int n=audio.read(buffer,0,buffer.length,AudioRecord.READ_BLOCKING);
                    if(n<=0)continue;
                    WebSocket ws=socket;
                    if(ws==null)break;
                    if(ws.queueSize()>512*1024)continue; // avoid stale audio build-up
                    byte[] chunk=new byte[n];
                    System.arraycopy(buffer,0,chunk,0,n);
                    String b64=Base64.encodeToString(chunk,Base64.NO_WRAP);
                    JSONObject data=new JSONObject().put("realtimeInput",
                        new JSONObject().put("audio",new JSONObject()
                            .put("mimeType","audio/pcm;rate=16000")
                            .put("data",b64)));
                    if(!ws.send(data.toString()))break;
                }
            }catch(Exception ex){
                ui(()->{if(active)listener.failure("마이크 전송 오류. 다른 녹음 앱을 종료하거나 권한을 확인하세요.");});
            }finally{
                if(audio!=null){
                    try{audio.stop();}catch(Exception ignored){}
                    try{audio.release();}catch(Exception ignored){}
                }
                recorder=null;
            }
        },"231-GeminiLiveMic").start();
    }
    public void stop(){
        active=false;ready=false;
        AudioRecord mic=recorder;
        if(mic!=null){try{mic.stop();}catch(Exception ignored){}}
        WebSocket ws=socket;
        socket=null;
        if(ws!=null){
            try{ws.send("{\"realtimeInput\":{\"audioStreamEnd\":true}}");}catch(Exception ignored){}
            try{ws.close(1000,"User stopped");}catch(Exception ignored){ws.cancel();}
        }
        client.dispatcher().cancelAll();client.connectionPool().evictAll();
    }
}
