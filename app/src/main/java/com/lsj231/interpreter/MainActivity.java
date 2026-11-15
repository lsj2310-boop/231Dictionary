package com.lsj231.interpreter;

import android.Manifest;
import android.app.Activity;
import android.app.AlertDialog;
import android.content.ClipData;
import android.net.Uri;
import android.os.SystemClock;
import android.text.InputType;
import android.widget.EditText;
import android.widget.CheckBox;
import android.content.ClipboardManager;
import android.content.ComponentName;
import android.content.ContentValues;
import android.content.Intent;
import android.content.SharedPreferences;
import android.content.pm.PackageManager;
import android.content.pm.ResolveInfo;
import android.database.Cursor;
import android.database.sqlite.SQLiteDatabase;
import android.graphics.Color;
import android.graphics.Typeface;
import android.graphics.drawable.GradientDrawable;
import android.os.Build;
import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;
import android.speech.RecognitionListener;
import android.speech.RecognitionService;
import android.speech.RecognizerIntent;
import android.speech.SpeechRecognizer;
import android.text.TextUtils;
import android.view.Gravity;
import android.view.View;
import android.view.WindowManager;
import android.widget.ArrayAdapter;
import android.widget.Button;
import android.widget.LinearLayout;
import android.widget.ScrollView;
import android.widget.SeekBar;
import android.widget.Spinner;
import android.widget.Switch;
import android.widget.TextView;
import android.widget.AdapterView;

import com.google.mlkit.common.model.DownloadConditions;
import com.google.mlkit.nl.translate.TranslateLanguage;
import com.google.mlkit.nl.translate.Translation;
import com.google.mlkit.nl.translate.Translator;
import com.google.mlkit.nl.translate.TranslatorOptions;

import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Date;
import java.util.List;
import java.util.Locale;

public final class MainActivity extends Activity implements RecognitionListener {
    private static final String[] NAMES = {"한국어", "English", "日本語", "中文", "Español", "Français", "Deutsch", "Italiano", "Português", "Tiếng Việt"};
    private static final String[] CODES = {"ko", "en", "ja", "zh", "es", "fr", "de", "it", "pt", "vi"};
    private static final String[] LOCALES = {"ko-KR", "en-US", "ja-JP", "zh-CN", "es-ES", "fr-FR", "de-DE", "it-IT", "pt-PT", "vi-VN"};
    private static final int MIC_REQUEST = 231;
    private static final int NAVY = Color.rgb(11,16,35);
    private static final int PURPLE = Color.rgb(105,69,246);
    private static final int GREEN = Color.rgb(42,195,144);
    private final Handler handler = new Handler(Looper.getMainLooper());
    private final ArrayList<Phrase> phrases = new ArrayList<>();
    private final ArrayList<ComponentName> engines = new ArrayList<>();
    private final ArrayList<String> engineLabels = new ArrayList<>();
    private SharedPreferences prefs;
    private SQLiteDatabase database;
    private LinearLayout root;
    private TextView statusView, sourceView, targetView, sourceTitle, targetTitle, hintView;
    private ScrollView sourceScroll, targetScroll;
    private Spinner fromSpinner, toSpinner;
    private Button micButton;
    private SpeechRecognizer recognizer;
    private Translator translator;
    private int modelEpoch=0, partialEpoch=0, sourceIndex=1, targetIndex=0, fontSp=21;
    private boolean running=false, translationReady=false, dark=true, offlinePreferred=false;
    private boolean installingModel=false, updateSpinners=false;
    private String partialSource="", partialTarget="";
    private String statusMessage="시작 버튼을 누르세요";
    private long lastResultAt=0;
    private long sessionId=-1;
    private Runnable partialJob;
    private ApiKeyVault apiVault;
    private GeminiLiveClient liveGemini;
    private GeminiTextTranslator geminiTranslator;
    private boolean geminiActive=false;
    private long lastPartialRequestAt=0;

    private static final class Phrase {
        String source;
        String target="";
        Phrase(String t) { source=t; }
    }

    @Override public void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        prefs=getSharedPreferences("231_settings",MODE_PRIVATE);
        apiVault=new ApiKeyVault(this);
        geminiTranslator=new GeminiTextTranslator();
        fontSp=prefs.getInt("font",21);
        dark=prefs.getBoolean("dark",true);
        offlinePreferred=prefs.getBoolean("offline",false);
        sourceIndex=prefs.getInt("from",1);
        targetIndex=prefs.getInt("to",0);
        database=openOrCreateDatabase("interpreter_history.db",MODE_PRIVATE,null);
        database.execSQL("CREATE TABLE IF NOT EXISTS history (id INTEGER PRIMARY KEY AUTOINCREMENT, created TEXT NOT NULL, body TEXT NOT NULL)");
        renderUI();
        initTranslation();
    }

    private int bg(){return dark?NAVY:Color.rgb(244,246,253);}
    private int panelBg(){return dark?Color.rgb(22,29,52):Color.WHITE;}
    private int fg(){return dark?Color.WHITE:Color.rgb(24,31,57);}
    private int subtle(){return dark?Color.rgb(160,174,204):Color.rgb(98,109,136);}
    private int outline(){return dark?Color.rgb(43,54,84):Color.rgb(220,225,238);}
    private int dp(float v){return (int)(v*getResources().getDisplayMetrics().density+0.5f);}
    private GradientDrawable round(int fill,int stroke,int radius) {
        GradientDrawable d=new GradientDrawable();
        d.setColor(fill);
        d.setCornerRadius(dp(radius));
        if(stroke!=Color.TRANSPARENT)d.setStroke(dp(1),stroke);
        return d;
    }
    private TextView text(String value,int sp,int color,boolean bold) {
        TextView t=new TextView(this);
        t.setText(value);t.setTextSize(sp);t.setTextColor(color);
        if(bold)t.setTypeface(Typeface.DEFAULT,Typeface.BOLD);
        return t;
    }
    private Button button(String value,int color) {
        Button b=new Button(this);
        b.setText(value);b.setTextSize(14);b.setAllCaps(false);b.setTextColor(Color.WHITE);
        b.setBackground(round(color,Color.TRANSPARENT,14));b.setMinHeight(0);
        b.setPadding(dp(10),dp(9),dp(10),dp(9));
        return b;
    }
    private LinearLayout row(){
        LinearLayout r=new LinearLayout(this);r.setOrientation(LinearLayout.HORIZONTAL);r.setGravity(Gravity.CENTER_VERTICAL);return r;
    }
    private LinearLayout.LayoutParams weight(int v){
        return new LinearLayout.LayoutParams(0,-1,v);
    }
    private Spinner spinner() {
        Spinner s=new Spinner(this);
        ArrayAdapter<String> a=new ArrayAdapter<String>(this,android.R.layout.simple_spinner_item,NAMES){
            @Override public View getView(int position,View convert,android.view.ViewGroup parent){
                TextView t=text(NAMES[position],15,fg(),true);
                t.setGravity(Gravity.CENTER_VERTICAL);t.setPadding(dp(12),dp(12),dp(3),dp(12));
                return t;
            }
            @Override public View getDropDownView(int position,View convert,android.view.ViewGroup parent){
                TextView t=text(NAMES[position],15,Color.rgb(25,27,40),false);
                t.setGravity(Gravity.CENTER_VERTICAL);t.setPadding(dp(18),dp(12),dp(12),dp(12));
                t.setBackgroundColor(Color.WHITE);
                return t;
            }
        };
        s.setAdapter(a);s.setBackground(round(panelBg(),outline(),12));
        return s;
    }
    private void renderUI() {
        root=new LinearLayout(this);root.setOrientation(LinearLayout.VERTICAL);
        root.setPadding(dp(16),dp(20),dp(16),dp(14));root.setBackgroundColor(bg());
        setContentView(root);
        if(prefs.getBoolean("awake",true)) getWindow().addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON);
        else getWindow().clearFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON);

        LinearLayout header=row();
        TextView title=text("231통역기",25,fg(),true);
        header.addView(title,new LinearLayout.LayoutParams(0,dp(56),1));
        title.setGravity(Gravity.CENTER_VERTICAL);
        Button newBtn=button("＋",panelBg());
        newBtn.setTextColor(fg());newBtn.setTextSize(19);
        header.addView(newBtn,new LinearLayout.LayoutParams(dp(45),dp(44)));
        newBtn.setOnClickListener(v->newConversation());
        Button settings=button("⚙",panelBg());settings.setTextColor(fg());settings.setTextSize(20);
        LinearLayout.LayoutParams hp=new LinearLayout.LayoutParams(dp(45),dp(44));hp.leftMargin=dp(8);
        header.addView(settings,hp);settings.setOnClickListener(v->showSettings());
        root.addView(header);
        Button geminiButton=button("Gemini",panelBg());
        geminiButton.setTextColor(PURPLE);
        geminiButton.setTextSize(12);
        LinearLayout.LayoutParams geminiLp=new LinearLayout.LayoutParams(dp(74),dp(44));
        geminiLp.leftMargin=dp(8);
        header.addView(geminiButton,geminiLp);
        geminiButton.setOnClickListener(v->showGeminiKeyDialog());

        statusView=text(statusMessage,12,GREEN,false);
        statusView.setGravity(Gravity.CENTER_VERTICAL);statusView.setPadding(dp(2),0,0,dp(10));
        root.addView(statusView,new LinearLayout.LayoutParams(-1,dp(29)));

        LinearLayout picker=row();picker.setPadding(0,0,0,dp(12));
        fromSpinner=spinner();toSpinner=spinner();
        picker.addView(fromSpinner,new LinearLayout.LayoutParams(0,dp(49),1));
        Button swap=button("⇄",PURPLE);swap.setTextSize(23);
        LinearLayout.LayoutParams swapLp=new LinearLayout.LayoutParams(dp(53),dp(48));swapLp.setMargins(dp(8),0,dp(8),0);
        picker.addView(swap,swapLp);
        picker.addView(toSpinner,new LinearLayout.LayoutParams(0,dp(49),1));
        root.addView(picker);
        updateSpinners=true;
        fromSpinner.setSelection(sourceIndex);
        toSpinner.setSelection(targetIndex);
        updateSpinners=false;
        fromSpinner.setOnItemSelectedListener(new AdapterView.OnItemSelectedListener(){
            public void onItemSelected(AdapterView<?> p,View v,int pos,long id){ if(!updateSpinners){sourceIndex=pos;changedLanguage();}}
            public void onNothingSelected(AdapterView<?> p){}
        });
        toSpinner.setOnItemSelectedListener(new AdapterView.OnItemSelectedListener(){
            public void onItemSelected(AdapterView<?> p,View v,int pos,long id){ if(!updateSpinners){targetIndex=pos;changedLanguage();}}
            public void onNothingSelected(AdapterView<?> p){}
        });
        swap.setOnClickListener(v->{
            int n=sourceIndex;sourceIndex=targetIndex;targetIndex=n;
            updateSpinners=true;fromSpinner.setSelection(sourceIndex);toSpinner.setSelection(targetIndex);
            updateSpinners=false;changedLanguage();
        });

        LinearLayout upper=panel("원문  ·  "+NAMES[sourceIndex],true);
        LinearLayout lower=panel("번역  ·  "+NAMES[targetIndex],false);
        LinearLayout.LayoutParams cardLp=new LinearLayout.LayoutParams(-1,0,1);
        cardLp.bottomMargin=dp(10);root.addView(upper,cardLp);
        root.addView(lower,new LinearLayout.LayoutParams(-1,0,1));

        hintView=text("실시간 자막은 기기의 음성 인식 성능과 인터넷 상태에 따라 지연될 수 있습니다.",11,subtle(),false);
        hintView.setGravity(Gravity.CENTER);hintView.setPadding(dp(5),dp(12),dp(5),dp(10));
        root.addView(hintView);

        LinearLayout footer=row();
        Button history=button("기록",panelBg());history.setTextColor(fg());
        footer.addView(history,new LinearLayout.LayoutParams(0,dp(54),1));
        micButton=button(running?"Ⅱ  일시정지":"●  통역 시작",running?Color.rgb(205,81,92):PURPLE);
        LinearLayout.LayoutParams mlp=new LinearLayout.LayoutParams(0,dp(54),2);
        mlp.setMargins(dp(8),0,dp(8),0);
        footer.addView(micButton,mlp);
        Button share=button("공유",panelBg());share.setTextColor(fg());
        footer.addView(share,new LinearLayout.LayoutParams(0,dp(54),1));
        root.addView(footer);
        history.setOnClickListener(v->showHistory());
        share.setOnClickListener(v->shareText());
        micButton.setOnClickListener(v->{if(running)stopListening();else startListening();});
        showTranscript();
        refreshLabels();
    }
    private LinearLayout panel(String label,boolean original) {
        LinearLayout card=new LinearLayout(this);card.setOrientation(LinearLayout.VERTICAL);
        card.setPadding(dp(16),dp(14),dp(16),dp(12));
        card.setBackground(round(panelBg(),outline(),18));
        LinearLayout labelRow=row();
        TextView dot=text(original?"●":"◆",11,original?GREEN:PURPLE,true);
        labelRow.addView(dot);
        TextView t=text("  "+label,14,subtle(),true);
        labelRow.addView(t);
        if(original)sourceTitle=t;else targetTitle=t;
        card.addView(labelRow,new LinearLayout.LayoutParams(-1,dp(27)));
        ScrollView scroll=new ScrollView(this);scroll.setFillViewport(true);
        TextView content=text("",fontSp,fg(),false);
        content.setLineSpacing(dp(5),1.05f);
        content.setTextIsSelectable(true);
        content.setPadding(0,dp(8),0,dp(4));
        scroll.addView(content,new ScrollView.LayoutParams(-1,-2));
        card.addView(scroll,new LinearLayout.LayoutParams(-1,0,1));
        if(original){sourceView=content;sourceScroll=scroll;}
        else{targetView=content;targetScroll=scroll;}
        return card;
    }
    private void refreshLabels(){
        if(sourceTitle!=null)sourceTitle.setText("  원문  ·  "+NAMES[sourceIndex]);
        if(targetTitle!=null)targetTitle.setText("  번역  ·  "+NAMES[targetIndex]);
    }
    private void status(String msg) {statusMessage=msg;if(statusView!=null)statusView.setText(msg);}
    private void changedLanguage() {
        prefs.edit().putInt("from",sourceIndex).putInt("to",targetIndex).apply();
        refreshLabels();
        boolean wasRunning=running;
        if(wasRunning)stopListening();
        partialSource="";partialTarget="";
        initTranslation();
        showTranscript();
        if(wasRunning)startListening();
    }
    private void initTranslation() {
        final int epoch=++modelEpoch;
        translationReady=false;installingModel=false;
        if(translator!=null){translator.close();translator=null;}
        if(sourceIndex==targetIndex){translationReady=true;status("같은 언어 · 음성을 그대로 표시합니다");return;}
        final String source=TranslateLanguage.fromLanguageTag(CODES[sourceIndex]);
        final String target=TranslateLanguage.fromLanguageTag(CODES[targetIndex]);
        if(source==null||target==null){status("선택한 언어는 번역을 지원하지 않습니다");return;}
        TranslatorOptions options=new TranslatorOptions.Builder().setSourceLanguage(source).setTargetLanguage(target).build();
        final Translator t=Translation.getClient(options);
        translator=t;
        installingModel=true;
        status("번역 모델 확인 중 · 최초 사용 시 다운로드");
        t.downloadModelIfNeeded(new DownloadConditions.Builder().build())
          .addOnSuccessListener(v->{
              if(epoch!=modelEpoch)return;
              translationReady=true;installingModel=false;
              status(running?"● 음성 인식 중 · 번역 준비됨":"번역 준비됨 · 통역 시작을 누르세요");
              for(Phrase p:new ArrayList<>(phrases))if(p.target.isEmpty())translateFinal(p);
              if(!partialSource.isEmpty())schedulePartial(partialSource);
          })
          .addOnFailureListener(e->{
              if(epoch!=modelEpoch)return;
              installingModel=false;translationReady=false;
              status("번역 모델 연결 실패 · 인터넷 확인 후 설정에서 재시도");
              if(!running)new AlertDialog.Builder(this).setTitle("번역 모델 준비 실패")
                  .setMessage("처음 선택한 언어는 Google ML Kit 번역 모델 다운로드가 필요합니다. Wi-Fi/모바일 데이터를 확인하고 설정의 '번역 모델 다시 준비'를 누르세요.\n\n"+e.getMessage())
                  .setPositiveButton("확인",null).show();
          });
    }
    private void startListening() {
        if(checkSelfPermission(Manifest.permission.RECORD_AUDIO)!=PackageManager.PERMISSION_GRANTED) {
            requestPermissions(new String[]{Manifest.permission.RECORD_AUDIO},MIC_REQUEST);return;
        }
        if(prefs.getBoolean("gemini_enabled",true)){
            String key=apiVault.read();
            if(key.isEmpty()){showGeminiKeyDialog();return;}
            startGemini(key);return;
        }
        if(!SpeechRecognizer.isRecognitionAvailable(this)) {
            new AlertDialog.Builder(this).setTitle("음성 인식 서비스를 찾을 수 없습니다")
            .setMessage("Google 음성 인식 서비스 또는 기기에 설치된 음성 인식 앱을 활성화하세요.")
            .setPositiveButton("확인",null).show();return;
        }
        stopRecognizerOnly();
        try {
            if(offlinePreferred&&Build.VERSION.SDK_INT>=31&&SpeechRecognizer.isOnDeviceRecognitionAvailable(this))
                recognizer=SpeechRecognizer.createOnDeviceSpeechRecognizer(this);
            else {
                String pkg=prefs.getString("engine_pkg","");
                String cls=prefs.getString("engine_cls","");
                if(!pkg.isEmpty()&&!cls.isEmpty())recognizer=SpeechRecognizer.createSpeechRecognizer(this,new ComponentName(pkg,cls));
                else recognizer=SpeechRecognizer.createSpeechRecognizer(this);
            }
            recognizer.setRecognitionListener(this);
            running=true;lastResultAt=0;
            micButton.setText("Ⅱ  일시정지");
            micButton.setBackground(round(Color.rgb(205,81,92),Color.TRANSPARENT,14));
            status("● 듣고 있습니다 · "+NAMES[sourceIndex]);
            listenOnce();
        }catch(Exception e){
            running=false;stopRecognizerOnly();status("음성 인식 초기화 실패: "+e.getMessage());
        }
    }

    private void showGeminiKeyDialog(){
        LinearLayout layout=new LinearLayout(this);layout.setOrientation(LinearLayout.VERTICAL);
        layout.setPadding(dp(22),dp(8),dp(22),dp(8));
        TextView info=text("Gemini 3.5 Transcribe Live\n현재 API 키: "+
            (apiVault.exists()?"등록됨 (암호화 보관)":"미등록")+
            "\nGoogle AI Studio에서 발급받은 개인 API 키를 입력하세요.\n마이크 음성이 Google API로 전송됩니다.",13,Color.DKGRAY,false);
        layout.addView(info);
        EditText field=new EditText(this);
        field.setSingleLine(true);
        field.setHint("Gemini API Key 입력");
        field.setInputType(InputType.TYPE_CLASS_TEXT|InputType.TYPE_TEXT_VARIATION_PASSWORD);
        layout.addView(field);
        new AlertDialog.Builder(this).setTitle("Gemini API 키")
           .setView(layout)
           .setPositiveButton("저장",(d,w)->{
               String key=field.getText().toString().trim();
               if(key.isEmpty()){status("변경 사항이 없습니다");return;}
               if(apiVault.save(key)){
                   prefs.edit().putBoolean("gemini_enabled",true).apply();
                   status("Gemini API 키 등록 완료 · 통역 시작을 누르세요");
               }else status("API 키 암호화 저장 실패");
           })
           .setNeutralButton("API 키 발급",(d,w)->{
               try{startActivity(new Intent(Intent.ACTION_VIEW,Uri.parse("https://aistudio.google.com/apikey")));}
               catch(Exception e){status("브라우저를 실행할 수 없습니다");}
           })
           .setNegativeButton("취소",null).show();
    }
    private void startGemini(String apiKey){
        stopRecognizerOnly();
        if(liveGemini!=null){liveGemini.stop();liveGemini=null;}
        running=true;geminiActive=true;
        partialEpoch++;partialSource="";partialTarget="";
        micButton.setText("Ⅱ  일시정지");
        micButton.setBackground(round(Color.rgb(205,81,92),Color.TRANSPARENT,14));
        status("Gemini Live 연결 중…");
        final GeminiLiveClient[] holder=new GeminiLiveClient[1];
        GeminiLiveClient session=new GeminiLiveClient(new GeminiLiveClient.Listener(){
            @Override public void status(String message){
                if(running&&geminiActive&&liveGemini==holder[0])
                    MainActivity.this.status(message);
            }
            @Override public void interim(String message){
                if(!running||!geminiActive||liveGemini!=holder[0])return;
                if(message.equals(partialSource))return;
                partialSource=message;partialTarget="";
                showTranscript();schedulePartial(message);
            }
            @Override public void committed(String message){
                if(!running||!geminiActive||liveGemini!=holder[0])return;
                if(message.isEmpty())return;
                Phrase item=new Phrase(message);
                phrases.add(item);lastResultAt=System.currentTimeMillis();
                partialEpoch++;partialSource="";partialTarget="";
                showTranscript();translateFinal(item);
            }
            @Override public void failure(String message){
                if(!running||!geminiActive||liveGemini!=holder[0])return;
                stopListening();MainActivity.this.status(message);
                new AlertDialog.Builder(MainActivity.this)
                    .setTitle("Gemini 실시간 음성 인식")
                    .setMessage(message+"\n\nAPI 키, 네트워크 연결 및 사용량 한도를 확인하세요.")
                    .setPositiveButton("확인",null).show();
            }
        });
        holder[0]=session;liveGemini=session;
        session.connect(apiKey,LOCALES[sourceIndex]);
    }

    private void listenOnce(){
        if(!running||recognizer==null)return;
        try{
            Intent i=new Intent(RecognizerIntent.ACTION_RECOGNIZE_SPEECH);
            i.putExtra(RecognizerIntent.EXTRA_LANGUAGE_MODEL,RecognizerIntent.LANGUAGE_MODEL_FREE_FORM);
            i.putExtra(RecognizerIntent.EXTRA_LANGUAGE,LOCALES[sourceIndex]);
            i.putExtra(RecognizerIntent.EXTRA_LANGUAGE_PREFERENCE,LOCALES[sourceIndex]);
            i.putExtra(RecognizerIntent.EXTRA_PARTIAL_RESULTS,true);
            i.putExtra(RecognizerIntent.EXTRA_MAX_RESULTS,1);
            i.putExtra(RecognizerIntent.EXTRA_PREFER_OFFLINE,offlinePreferred);
            recognizer.startListening(i);
        }catch(Exception e){
            status("마이크 시작 오류: "+e.getMessage());
            running=false;
            if(micButton!=null)micButton.setText("●  통역 시작");
        }
    }
    private void stopRecognizerOnly(){
        if(recognizer!=null){
            try{recognizer.cancel();recognizer.destroy();}catch(Exception ignored){}
            recognizer=null;
        }
    }
    private void stopListening(){
        running=false;handler.removeCallbacksAndMessages(null);
        stopRecognizerOnly();
        geminiActive=false;
        if(liveGemini!=null){liveGemini.stop();liveGemini=null;}
        partialSource="";partialTarget="";
        if(micButton!=null){
            micButton.setText("●  통역 시작");
            micButton.setBackground(round(PURPLE,Color.TRANSPARENT,14));
        }
        status("일시정지 · 통역 시작을 누르면 재개됩니다");
        showTranscript();
    }
    private String best(Bundle b){
        if(b==null)return "";
        ArrayList<String> candidates=b.getStringArrayList(SpeechRecognizer.RESULTS_RECOGNITION);
        return candidates==null||candidates.isEmpty()?"":candidates.get(0).trim();
    }
    @Override public void onReadyForSpeech(Bundle b){if(running)status("● 듣고 있습니다 · "+NAMES[sourceIndex]);}
    @Override public void onBeginningOfSpeech(){if(running)status("● 말하는 내용을 인식 중…");}
    @Override public void onRmsChanged(float rmsdB){}
    @Override public void onBufferReceived(byte[] buffer){}
    @Override public void onEndOfSpeech(){if(running)status("음성 구간 처리 중…");}
    @Override public void onEvent(int eventType, Bundle params){}
    @Override public void onPartialResults(Bundle b) {
        if(!running)return;
        String text=best(b);
        if(text.isEmpty()||text.equals(partialSource))return;
        partialSource=text;partialTarget="";
        showTranscript();schedulePartial(text);
    }
    private void schedulePartial(String text){
        if(partialJob!=null)handler.removeCallbacks(partialJob);
        final int sequence=++partialEpoch;
        long wait=Math.max(80,650-(SystemClock.elapsedRealtime()-lastPartialRequestAt));
        partialJob=()->{
            if(!running||!text.equals(partialSource)||sequence!=partialEpoch)return;
            lastPartialRequestAt=SystemClock.elapsedRealtime();
            if(sourceIndex==targetIndex){partialTarget=text;showTranscript();return;}
            final Translator t=translator;
            if(translationReady&&t!=null){
                t.translate(text).addOnSuccessListener(translated->{
                    if(running&&sequence==partialEpoch&&t==translator&&text.equals(partialSource)){
                        partialTarget=translated;showTranscript();
                    }
                }).addOnFailureListener(e->{});
            }else if(geminiActive){
                String key=apiVault.read();
                if(!key.isEmpty()){
                    final int src=sourceIndex,dst=targetIndex;
                    geminiTranslator.translate(key,text,NAMES[src],NAMES[dst],(result,error)->runOnUiThread(()->{
                        if(result!=null&&running&&sequence==partialEpoch&&
                            src==sourceIndex&&dst==targetIndex&&text.equals(partialSource)){
                            partialTarget=result;showTranscript();
                        }
                    }));
                }
            }
        };
        handler.postDelayed(partialJob,wait);
    }
    @Override public void onResults(Bundle b){
        if(!running)return;
        String text=best(b);
        if(!text.isEmpty()&&(!text.equals(lastCommitted())||System.currentTimeMillis()-lastResultAt>2500)){
            Phrase p=new Phrase(text);phrases.add(p);
            lastResultAt=System.currentTimeMillis();translateFinal(p);
        }
        partialEpoch++;partialSource="";partialTarget="";
        showTranscript();
        if(running)handler.postDelayed(()->{if(running)listenOnce();},450);
    }
    private String lastCommitted(){
        return phrases.isEmpty()?"":phrases.get(phrases.size()-1).source;
    }
    @Override public void onError(int error){
        if(!running)return;
        partialEpoch++;partialSource="";partialTarget="";showTranscript();
        if(error==SpeechRecognizer.ERROR_INSUFFICIENT_PERMISSIONS) {
            stopListening();status("마이크 권한이 필요합니다");return;
        }
        if(error==SpeechRecognizer.ERROR_CLIENT&&lastResultAt==0){
            status("음성 인식 서비스 오류 · 설정에서 엔진을 확인하세요");
        }else{
            status("● 다음 말을 듣는 중…");
        }
        long delay=(error==SpeechRecognizer.ERROR_RECOGNIZER_BUSY||error==SpeechRecognizer.ERROR_CLIENT)?1350:700;
        handler.postDelayed(()->{if(running)listenOnce();},delay);
    }
    private void translateFinal(Phrase p){
        if(sourceIndex==targetIndex){p.target=p.source;showTranscript();return;}
        final String key=apiVault.read();
        if(!key.isEmpty()&&prefs.getBoolean("gemini_enabled",true)){
            final int src=sourceIndex,dst=targetIndex;
            geminiTranslator.translate(key,p.source,NAMES[src],NAMES[dst],(text,error)->runOnUiThread(()->{
                if(!phrases.contains(p)||src!=sourceIndex||dst!=targetIndex)return;
                if(text!=null&&!text.isEmpty()){
                    p.target=text;showTranscript();
                }else translateFinalLocal(p);
            }));
        }else translateFinalLocal(p);
    }
    private void translateFinalLocal(Phrase p){
        if(sourceIndex==targetIndex){p.target=p.source;showTranscript();return;}
        if(!translationReady||translator==null){
            p.target="[번역 준비 중 · 모델 다운로드 확인]";showTranscript();return;
        }
        final Translator t=translator;
        t.translate(p.source).addOnSuccessListener(text->{
            if(t!=translator||!phrases.contains(p))return;
            p.target=text;showTranscript();
        }).addOnFailureListener(e->{
            if(t==translator&&phrases.contains(p)){
                p.target="[번역 실패 · 인터넷/모델 확인]";
                showTranscript();
            }
        });
    }
    private void showTranscript(){
        if(sourceView==null||targetView==null)return;
        StringBuilder originals=new StringBuilder();
        StringBuilder translated=new StringBuilder();
        for(Phrase p:phrases){
            if(originals.length()>0){originals.append("\n\n");translated.append("\n\n");}
            originals.append(p.source);
            translated.append(p.target.isEmpty()?"… 번역 중":p.target);
        }
        if(!partialSource.isEmpty()){
            if(originals.length()>0){originals.append("\n\n");translated.append("\n\n");}
            originals.append(partialSource).append("  …");
            translated.append(partialTarget.isEmpty()?"… 번역 중":partialTarget).append("  …");
        }
        if(originals.length()==0)originals.append("마이크 버튼을 누르고 말을 시작하세요.\n\n영어 음성을 영어 원문으로,\n아래에는 한국어 번역으로 표시합니다.");
        if(translated.length()==0)translated.append("여기에 번역문이 실시간 표시됩니다.\n\n영어 ↔ 한국어 등 언어 방향을 바꿀 수 있습니다.");
        sourceView.setText(originals.toString());targetView.setText(translated.toString());
        sourceView.setTextSize(fontSp);targetView.setTextSize(fontSp);
        sourceScroll.post(()->sourceScroll.fullScroll(View.FOCUS_DOWN));
        targetScroll.post(()->targetScroll.fullScroll(View.FOCUS_DOWN));
    }
    private String compiled(){
        StringBuilder sb=new StringBuilder();
        sb.append("231통역기 대화 기록\n");
        sb.append(NAMES[sourceIndex]).append(" → ").append(NAMES[targetIndex]).append("\n");
        sb.append(new SimpleDateFormat("yyyy-MM-dd HH:mm",Locale.KOREA).format(new Date())).append("\n\n");
        for(Phrase p:phrases)sb.append("원문: ").append(p.source).append("\n번역: ").append(p.target).append("\n\n");
        return sb.toString();
    }
    private void saveCurrent(){
        if(phrases.isEmpty())return;
        ContentValues data=new ContentValues();
        data.put("created",new SimpleDateFormat("yyyy-MM-dd HH:mm",Locale.KOREA).format(new Date()));
        data.put("body",compiled());
        if(sessionId<0)sessionId=database.insert("history",null,data);
        else database.update("history",data,"id=?",new String[]{String.valueOf(sessionId)});
        status("대화 기록이 휴대폰에 저장되었습니다");
    }
    private void newConversation(){
        if(phrases.isEmpty())return;
        new AlertDialog.Builder(this).setTitle("새 대화")
        .setMessage("현재 내용을 휴대폰에 저장하고 새 대화를 시작할까요?")
        .setPositiveButton("저장 후 시작",(d,w)->{
            saveCurrent();phrases.clear();sessionId=-1;partialSource="";partialTarget="";showTranscript();
        }).setNegativeButton("취소",null).show();
    }
    private void shareText(){
        if(phrases.isEmpty()){status("공유할 대화가 없습니다");return;}
        saveCurrent();
        Intent i=new Intent(Intent.ACTION_SEND);
        i.setType("text/plain");i.putExtra(Intent.EXTRA_SUBJECT,"231통역기 대화 기록");i.putExtra(Intent.EXTRA_TEXT,compiled());
        startActivity(Intent.createChooser(i,"대화 기록 공유 / TXT 내용 전송"));
    }
    private void showHistory(){
        saveCurrent();
        ArrayList<Long> ids=new ArrayList<>();
        ArrayList<String> full=new ArrayList<>();
        ArrayList<String> labels=new ArrayList<>();
        try(Cursor c=database.rawQuery("SELECT id,created,body FROM history ORDER BY id DESC LIMIT 60",null)){
            while(c.moveToNext()){
                ids.add(c.getLong(0));full.add(c.getString(2));
                String s=c.getString(2).replace("\n"," ");
                labels.add(c.getString(1)+"\n"+s.substring(0,Math.min(85,s.length())));
            }
        }
        if(labels.isEmpty()){new AlertDialog.Builder(this).setTitle("대화 기록").setMessage("저장된 대화가 없습니다.").setPositiveButton("확인",null).show();return;}
        new AlertDialog.Builder(this).setTitle("내 대화 기록 (기기에만 저장)")
          .setItems(labels.toArray(new String[0]),(d,which)->historyActions(ids.get(which),full.get(which)))
          .setNegativeButton("닫기",null).show();
    }
    private void historyActions(long id,String value){
        String[] options={"내용 보기","텍스트 복사","다른 앱으로 공유","이 기록 삭제"};
        new AlertDialog.Builder(this).setTitle("대화 기록")
          .setItems(options,(d,which)->{
              if(which==0){
                  ScrollView scroll=new ScrollView(this);TextView t=text(value,16,Color.BLACK,false);
                  t.setPadding(dp(18),dp(12),dp(18),dp(12));t.setTextIsSelectable(true);
                  scroll.addView(t);
                  new AlertDialog.Builder(this).setTitle("대화 내용").setView(scroll).setPositiveButton("확인",null).show();
              }else if(which==1){
                  ClipboardManager cm=(ClipboardManager)getSystemService(CLIPBOARD_SERVICE);
                  cm.setPrimaryClip(ClipData.newPlainText("231통역기",value));status("클립보드로 복사되었습니다");
              }else if(which==2){
                  Intent i=new Intent(Intent.ACTION_SEND);i.setType("text/plain");
                  i.putExtra(Intent.EXTRA_TEXT,value);startActivity(Intent.createChooser(i,"기록 공유"));
              }else {
                  new AlertDialog.Builder(this).setMessage("이 대화 기록을 삭제할까요?")
                    .setPositiveButton("삭제",(dialog,w)->{
                        database.delete("history","id=?",new String[]{String.valueOf(id)});
                        if(sessionId==id)sessionId=-1;
                        status("기록을 삭제했습니다");
                    }).setNegativeButton("취소",null).show();
              }
          }).setNegativeButton("닫기",null).show();
    }
    private void engineList(){
        engines.clear();engineLabels.clear();
        engines.add(null);engineLabels.add("휴대폰 시스템 기본 (권장)");
        try {
            PackageManager pm=getPackageManager();
            Intent i=new Intent(RecognitionService.SERVICE_INTERFACE);
            List<ResolveInfo> services=pm.queryIntentServices(i,PackageManager.GET_META_DATA);
            for(ResolveInfo info:services){
                if(info.serviceInfo==null)continue;
                engines.add(new ComponentName(info.serviceInfo.packageName,info.serviceInfo.name));
                CharSequence label=info.loadLabel(pm);
                engineLabels.add((label==null?info.serviceInfo.packageName:label.toString())
                   +"  ("+info.serviceInfo.packageName+")");
            }
        }catch(Exception ignored){}
    }
    private void showSettings(){
        engineList();
        ScrollView sc=new ScrollView(this);
        LinearLayout l=new LinearLayout(this);l.setOrientation(LinearLayout.VERTICAL);
        l.setPadding(dp(20),dp(8),dp(20),dp(12));sc.addView(l);
        TextView heading=text("Gemini Live 음성 인식",17,PURPLE,true);
        l.addView(heading);
        TextView current=text("API 키: "+(apiVault.exists()?"● 등록됨":"○ 미등록")+
             "\n음성: Gemini 3.5 Transcribe Live\n번역: Gemini 3.5 Flash-Lite",13,Color.DKGRAY,false);
        current.setPadding(0,dp(8),0,dp(8));l.addView(current);
        Switch geminiSwitch=new Switch(this);
        geminiSwitch.setText("Gemini Live 실시간 인식 사용 (권장)");
        geminiSwitch.setChecked(prefs.getBoolean("gemini_enabled",true));
        l.addView(geminiSwitch);
        TextView keyLabel=text("Gemini API 키 등록/변경",14,Color.DKGRAY,true);
        keyLabel.setPadding(0,dp(12),0,0);l.addView(keyLabel);
        EditText apiField=new EditText(this);
        apiField.setSingleLine(true);apiField.setTextSize(14);
        apiField.setHint("새 API 키 입력 (빈 칸이면 기존 키 유지)");
        apiField.setInputType(InputType.TYPE_CLASS_TEXT|InputType.TYPE_TEXT_VARIATION_PASSWORD);
        l.addView(apiField);
        CheckBox deleteKey=new CheckBox(this);
        deleteKey.setText("현재 저장된 Gemini API 키 삭제");
        l.addView(deleteKey);
        TextView warning=text("Gemini 서버로 음성이 전송됩니다. 인터넷 및 본인 API 키가 필요하며 사용량에 따라 요금이 발생할 수 있습니다.",12,Color.GRAY,false);
        warning.setPadding(0,0,0,dp(16));l.addView(warning);
        TextView sizeLabel=text("자막 글자 크기  ·  "+fontSp+"sp",16,Color.DKGRAY,true);
        l.addView(sizeLabel);
        SeekBar size=new SeekBar(this);size.setMax(25);size.setProgress(fontSp-15);
        size.setOnSeekBarChangeListener(new SeekBar.OnSeekBarChangeListener(){
            public void onProgressChanged(SeekBar b,int n,boolean fromUser){
                fontSp=15+n;sizeLabel.setText("자막 글자 크기  ·  "+fontSp+"sp");
                if(sourceView!=null)sourceView.setTextSize(fontSp);
                if(targetView!=null)targetView.setTextSize(fontSp);
                prefs.edit().putInt("font",fontSp).apply();
            }
            public void onStartTrackingTouch(SeekBar b){}
            public void onStopTrackingTouch(SeekBar b){}
        });
        l.addView(size);
        Switch theme=new Switch(this);theme.setText("어두운 화면 (다크 모드)");theme.setChecked(dark);l.addView(theme);
        Switch awake=new Switch(this);awake.setText("통역 중 화면 켜두기");awake.setChecked(prefs.getBoolean("awake",true));l.addView(awake);
        Switch offline=new Switch(this);offline.setText("기기 내 음성 인식 우선 (지원 시)");offline.setChecked(offlinePreferred);l.addView(offline);
        TextView description=text("· 음성 인식: 갤럭시 기본/Google 등 설치된 시스템 서비스\n· 번역: Google ML Kit 기기 내 번역\n· 처음 사용하는 번역 언어는 모델 다운로드 필요\n· 음성 인식은 선택한 서비스에 따라 인터넷을 사용할 수 있습니다.",13,Color.DKGRAY,false);
        description.setPadding(0,dp(14),0,dp(12));l.addView(description);
        TextView engineTitle=text("음성 인식 엔진",16,Color.DKGRAY,true);l.addView(engineTitle);
        Spinner engineSpinner=new Spinner(this);
        ArrayAdapter<String> engineAdapter=new ArrayAdapter<>(this,android.R.layout.simple_spinner_dropdown_item,engineLabels);
        engineSpinner.setAdapter(engineAdapter);l.addView(engineSpinner);
        int selected=0;
        String pkg=prefs.getString("engine_pkg",""), cls=prefs.getString("engine_cls","");
        for(int k=1;k<engines.size();k++){
            ComponentName c=engines.get(k);
            if(c.getPackageName().equals(pkg)&&c.getClassName().equals(cls)){selected=k;break;}
        }
        engineSpinner.setSelection(selected);
        TextView engineHelp=text("즐겨찾기: 이 화면에서 선택한 엔진이 다음 실행부터 기본 엔진으로 사용됩니다.",12,Color.GRAY,false);
        engineHelp.setPadding(0,dp(9),0,dp(6));l.addView(engineHelp);
        new AlertDialog.Builder(this).setTitle("231통역기 설정").setView(sc)
         .setNeutralButton("번역 모델 다시 준비",(d,w)->{
             prefs.edit().putInt("font",fontSp).apply();initTranslation();
         })
         .setPositiveButton("설정 저장",(d,w)->{
             if(deleteKey.isChecked())apiVault.clear();
             else if(!apiField.getText().toString().trim().isEmpty()){
                 if(!apiVault.save(apiField.getText().toString().trim()))status("Gemini 키 저장 실패");
             }
             prefs.edit().putBoolean("gemini_enabled",geminiSwitch.isChecked()).apply();
             boolean themeChanged=(dark!=theme.isChecked());
             dark=theme.isChecked();offlinePreferred=offline.isChecked();
             prefs.edit().putBoolean("dark",dark).putBoolean("offline",offlinePreferred)
               .putBoolean("awake",awake.isChecked()).putInt("font",fontSp).apply();
             int chosen=engineSpinner.getSelectedItemPosition();
             ComponentName chosenEngine=chosen>=0&&chosen<engines.size()?engines.get(chosen):null;
             prefs.edit().putString("engine_pkg",chosenEngine==null?"":chosenEngine.getPackageName())
                 .putString("engine_cls",chosenEngine==null?"":chosenEngine.getClassName()).apply();
             boolean wasRunning=running;
             if(wasRunning)stopListening();
             if(themeChanged)renderUI();
             if(wasRunning)startListening();
         }).setNegativeButton("취소",null).show();
    }
    @Override public void onRequestPermissionsResult(int requestCode,String[] permissions,int[] grantResults){
        super.onRequestPermissionsResult(requestCode,permissions,grantResults);
        if(requestCode==MIC_REQUEST){
            if(grantResults.length>0&&grantResults[0]==PackageManager.PERMISSION_GRANTED)startListening();
            else status("마이크 사용 권한이 필요합니다");
        }
    }
    @Override protected void onStop(){
        super.onStop();
        if(running)stopListening();
        saveCurrent();
    }
    @Override protected void onDestroy(){
        running=false;handler.removeCallbacksAndMessages(null);
        stopRecognizerOnly();
        if(liveGemini!=null){liveGemini.stop();liveGemini=null;}
        if(geminiTranslator!=null)geminiTranslator.close();
        if(translator!=null){translator.close();translator=null;}
        if(database!=null)database.close();
        super.onDestroy();
    }
}
