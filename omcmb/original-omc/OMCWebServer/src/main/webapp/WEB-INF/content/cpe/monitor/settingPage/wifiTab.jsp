<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
    #cpeWifiConfigPage{
        height: 100%;
        width: 100%;
    }
    #cpeWifiConfigPage .itemMainBoxCls{
        border-radius:10px;
        background:#fff;
        height:100%;
        width: 100%;
        display: flex;
        flex-direction: column;
        position: relative;
    }
    #cpeWifiConfigPage .itemMainBoxTitle {
        height:36px;
        padding-left: 20px;
        line-height: 36px;
        font-size:14px;
        font-weight:bold;
        border-bottom: 1px solid #E9E9E9;
    }
    #cpeWifiConfigPage .itemMainBoxCenter{
        width: 100%;
        flex:1;
        overflow: auto;
    }
    #cpeWifiConfigPage .itemMainBoxFooter{
        display: flex;
        align-items: center;
        border-top : 1px solid #E9E9E9;
        height:48px;
        background-color: #FFFFFF;
        box-sizing: border-box;
        width: 100%;
        padding-left: 20px;
    }
    #cpeWifiConfigPage .el-form-item{
        margin-bottom: 20px;
    }
    #cpeWifiConfigPage .el-form-item .el-form-item__label{
        font-size: 12px;
    }
    #cpeWifiConfigPage .paramsItemBoxCls{
        display: flex;
        align-items: center;
        margin-bottom: 20px;
        min-width: 740px;
    }
    #cpeWifiConfigPage .paramsItemBoxCls .el-form-item{
        margin-bottom: 0px;
    }
    #cpeWifiConfigPage .paramsItemLabelCls{
        width: 140px;
    }
    #cpeWifiConfigPage .exportBtnBoxCls{
        height: 28px;
        width: 130px;
        margin-left: 140px;
        border : 1px solid #E9E9E9;
        border-radius: 4px;
        font-size: 12px;
        display: flex;
        align-items: center;
        justify-content: center;
        cursor: pointer;
        margin-bottom: 20px;
    }
    #cpeWifiConfigPage .exportTemplateBoxCls{
        display: flex;
        align-items: center;
    }
    #cpeWifiConfigPage .exportTemplateTipCls{
        color: rgba(0,0,0,0.32);
        margin-right: 20px;
    }
    
    #cpeWifiConfigPage .validate-item .el-input__inner{
        width:200px;
    }
    #cpeWifiConfigPage .validate-item .el-input-group__append{
        border:none;
        background:none;
        padding: 0px 10px;
    }
    #cpeWifiConfigPage .validate-item .el-form-item__error{
        display:none;
    }
    #cpeWifiConfigPage .is-error .el-input-group__append{
        color:#FA5555;
    }
    #cpeWifiConfigPage .notSupportTipBoxCls{
        color: rgba(0,0,0,0.32);
        font-size: 14px;
        margin-left: 20px;
    }
    .paramsGroup {
        display: flex;
        flex-wrap: wrap;
    }
    .paramsGroup .el-form-item {
        width: 45%;
    }
    .paramsGroup .params-group-tile {
        width: 100%;
        font-size: 12px;
        font-weight: bold;
        padding-bottom: 10px;
    }
    .with-append-input {
        width: auto;
    }
    .with-append-input .el-input__inner {
        width: 200px;
    }
    .with-append-input .el-input-group__append {
        border: none;
        background: transparent;
    }
    .margin-offset {
        margin-left: -40px;
        margin-right: 18px;
    }
    .background-color-white {
        background-color: #fff;
    }
</style>

<div id="cpeWifiConfigPage">
	<div class="itemMainBoxCls">
        <div class="itemMainBoxTitle" style="position: relative;">
            WiFi config
            <span style="margin-left: 10px;color: #4D84FF;font-weight: normal;" v-if="basicForm.mrCombineEnable == 'NotSupport'">(Not support)</span>
            <span style="margin-left: 10px;color: #4D84FF;font-weight: normal;" v-if="basicForm.mrCombineEnable == 'NotSync'">(Not synchronized,please click "Refresh".)</span>
            <div style="position: absolute;right: 20px;top: 0px;z-index: 10;">
                <i class="el-icon el-icon-circle-refresh" @click="refreshWiFi"></i>
            </div>
        </div>
		<div :class="{'itemMainBoxCenter':true, 'loading': isRefresh}">
            <el-collapse v-model="activeCollapse">
                <el-collapse-item name="Basic">
                    <template slot='title'>
                        <p style="display:inline-block;margin-left:40px;">
                            <span style="font-size:14px;font-weight:bold">Basic Settings</span>
                        </p>
                    </template>
                    <div class="rightContentCls" >
                        <el-form :model='basicForm' ref="basicForm" :rules="rules" label-position="top">
                            <div class="paramsItemBoxCls">
                                <div class="paramsItemLabelCls">Dual frequency in one</div>
                                <el-form-item prop='mrCombineEnable' style="width:40%;min-width:400px;" label="">
                                    <el-switch v-model="basicForm.mrCombineEnable" active-value="1" inactive-value="0" :disabled="wifiDisabled"></el-switch>
                                </el-form-item>
                            </div>
                            <div class="paramsItemBoxCls">
                                <div class="paramsItemLabelCls">Guest</div>
                                <el-form-item prop='mrGuestEable' style="width:40%;min-width:400px;" label="">
                                    <el-switch v-model="basicForm.mrGuestEable" active-value="1" inactive-value="0" :disabled="wifiDisabled"></el-switch>
                                </el-form-item>
                            </div>
                            <!-- Master -->
                            <div class="paramsGroup" v-show="basicForm.mrCombineEnable == '1'">
                                <div class="params-group-tile">Master SSID</div>
                                <el-form-item label="SSID" prop="mr5MasterSsid">
                                    <el-input v-model="basicForm.mr5MasterSsid" class="with-append-input" maxlength="32" :disabled="wifiDisabled">
                                        <span slot="append">Length: 0-32</span>
                                    </el-input>
                                </el-form-item>
                                <el-form-item label="Encryption" prop="mr5MasterEncryption">
                                    <el-select v-model="basicForm.mr5MasterEncryption" :disabled="wifiDisabled">
                                        <el-option label="WPA(AES)-PSK" value="WPA"></el-option>
                                        <el-option label="WPA2(AES)-PSK" value="WPA2"></el-option>
                                    </el-select>
                                </el-form-item>
                                <el-form-item label="Password" prop="mr5MasterPassPhrase">
                                    <el-input :type="airMap.m5GOpen?'text':'password'" v-model="basicForm.mr5MasterPassPhrase" class="with-append-input" :disabled="wifiDisabled">
                                        <span slot="append">
                                            <i class="el-icon margin-offset" 
                                                :class="{'background-color-white': true,'el-icon-operation-hide': !airMap.m5GOpen, 'el-icon-operation-view': airMap.m5GOpen}" 
                                                @click="airMap.m5GOpen = !airMap.m5GOpen">
                                            </i>
                                            Range: 8-64 bit strings in [a-za-z0-9_-]
                                        </span>
                                    </el-input>
                                </el-form-item>
                            </div>
                            <!-- Guest -->
                            <div class="paramsGroup" v-show="basicForm.mrCombineEnable == '1' && basicForm.mrGuestEable == '1'">
                                <div class="params-group-tile">Guest SSID</div>
                                <el-form-item label="SSID" prop="mr5GuestSsid">
                                    <el-input v-model="basicForm.mr5GuestSsid" class="with-append-input" maxlength="32" :disabled="wifiDisabled">
                                        <span slot="append">Length: 0-32</span>
                                    </el-input>
                                </el-form-item>
                                <el-form-item label="Encryption" prop="mr5GuestEncryption">
                                    <el-select v-model="basicForm.mr5GuestEncryption" :disabled="wifiDisabled">
                                        <el-option label="WPA(AES)-PSK" value="WPA"></el-option>
                                        <el-option label="WPA2(AES)-PSK" value="WPA2"></el-option>
                                    </el-select>
                                </el-form-item>
                                <el-form-item label="Password" prop="mr5GuestPassPhrase">
                                    <el-input :type="airMap.g5GOpen?'text':'password'" v-model="basicForm.mr5GuestPassPhrase" class="with-append-input" :disabled="wifiDisabled">
                                        <span slot="append">
                                            <i class="el-icon margin-offset" 
                                                :class="{'background-color-white': true,'el-icon-operation-hide': !airMap.g5GOpen, 'el-icon-operation-view': airMap.g5GOpen}" 
                                                @click="airMap.g5GOpen = !airMap.g5GOpen">
                                            </i>
                                            Range: 8-64 bit strings in [a-za-z0-9_-]
                                        </span>
                                    </el-input>
                                </el-form-item>
                            </div>

                            <!-- Master 2.4G -->
                            <div class="paramsGroup" v-show="basicForm.mrCombineEnable == '0'">
                                <div class="params-group-tile">Master 2.4G SSID</div>
                                <el-form-item label="SSID" prop="mr4MasterSsid">
                                    <el-input v-model="basicForm.mr4MasterSsid" class="with-append-input" maxlength="32" :disabled="wifiDisabled">
                                        <span slot="append">Length: 0-32</span>
                                    </el-input>
                                </el-form-item>
                                <el-form-item label="Encryption" prop="mr4MasterEncryption">
                                    <el-select v-model="basicForm.mr4MasterEncryption" :disabled="wifiDisabled">
                                        <el-option label="WPA(AES)-PSK" value="WPA"></el-option>
                                        <el-option label="WPA2(AES)-PSK" value="WPA2"></el-option>
                                    </el-select>
                                </el-form-item>
                                <el-form-item label="Password" prop="mr4MasterPassPhrase">
                                    <el-input :type="airMap.m4GOpen?'text':'password'" v-model="basicForm.mr4MasterPassPhrase" class="with-append-input" :disabled="wifiDisabled">
                                        <span slot="append">
                                            <i class="el-icon margin-offset" 
                                                :class="{'background-color-white': true,'el-icon-operation-hide': !airMap.m4GOpen, 'el-icon-operation-view': airMap.m4GOpen}" 
                                                @click="airMap.m4GOpen = !airMap.m4GOpen">
                                            </i>
                                            Range: 8-64 bit strings in [a-za-z0-9_-]
                                        </span>
                                    </el-input>
                                </el-form-item>
                            </div>
                            <!-- Master 5G -->
                            <div class="paramsGroup" v-show="basicForm.mrCombineEnable == '0'">
                                <div class="params-group-tile">Master 5G SSID</div>
                                <el-form-item label="SSID" prop="mr5MasterSsid">
                                    <el-input v-model="basicForm.mr5MasterSsid" class="with-append-input" maxlength="32" :disabled="wifiDisabled">
                                        <span slot="append">Length: 0-32</span>
                                    </el-input>
                                </el-form-item>
                                <el-form-item label="Encryption" prop="mr5MasterEncryption">
                                    <el-select v-model="basicForm.mr5MasterEncryption" :disabled="wifiDisabled">
                                        <el-option label="WPA(AES)-PSK" value="WPA"></el-option>
                                        <el-option label="WPA2(AES)-PSK" value="WPA2"></el-option>
                                    </el-select>
                                </el-form-item>
                                <el-form-item label="Password" prop="mr5MasterPassPhrase">
                                    <el-input :type="airMap.m5GOpen?'text':'password'" v-model="basicForm.mr5MasterPassPhrase" class="with-append-input" :disabled="wifiDisabled">
                                        <span slot="append">
                                            <i class="el-icon margin-offset" 
                                                :class="{'background-color-white': true,'el-icon-operation-hide': !airMap.m5GOpen, 'el-icon-operation-view': airMap.m5GOpen}" 
                                                @click="airMap.m5GOpen = !airMap.m5GOpen">
                                            </i>
                                            Range: 8-64 bit strings in [a-za-z0-9_-]
                                        </span>
                                    </el-input>
                                </el-form-item>
                            </div>

                            <!-- Guest 2.4G -->
                            <div class="paramsGroup" v-show="basicForm.mrCombineEnable == '0' && basicForm.mrGuestEable == '1'">
                                <div class="params-group-tile">Guest 2.4G SSID</div>
                                <el-form-item label="SSID" prop="mr4GuestSsid">
                                    <el-input v-model="basicForm.mr4GuestSsid" class="with-append-input" maxlength="32" :disabled="wifiDisabled">
                                        <span slot="append">Length: 0-32</span>
                                    </el-input>
                                </el-form-item>
                                <el-form-item label="Encryption" prop="mr4GuestEncryption">
                                    <el-select v-model="basicForm.mr4GuestEncryption" :disabled="wifiDisabled">
                                        <el-option label="WPA(AES)-PSK" value="WPA"></el-option>
                                        <el-option label="WPA2(AES)-PSK" value="WPA2"></el-option>
                                    </el-select>
                                </el-form-item>
                                <el-form-item label="Password" prop="mr4GuestPassPhrase">
                                    <el-input :type="airMap.g4GOpen?'text':'password'" v-model="basicForm.mr4GuestPassPhrase" class="with-append-input" :disabled="wifiDisabled">
                                        <span slot="append">
                                            <i class="el-icon margin-offset" 
                                                :class="{'background-color-white': true,'el-icon-operation-hide': !airMap.g4GOpen, 'el-icon-operation-view': airMap.g4GOpen}" 
                                                @click="airMap.g4GOpen = !airMap.g4GOpen">
                                            </i>
                                            Range: 8-64 bit strings in [a-za-z0-9_-]
                                        </span>
                                    </el-input>
                                </el-form-item>
                            </div>
                            <!-- Guest 5G -->
                            <div class="paramsGroup" v-show="basicForm.mrCombineEnable == '0' && basicForm.mrGuestEable == '1'">
                                <div class="params-group-tile">Guest 5G SSID</div>
                                <el-form-item label="SSID" prop="mr5GuestSsid">
                                    <el-input v-model="basicForm.mr5GuestSsid" class="with-append-input" maxlength="32" :disabled="wifiDisabled">
                                        <span slot="append">Length: 0-32</span>
                                    </el-input>
                                </el-form-item>
                                <el-form-item label="Encryption" prop="mr5GuestEncryption">
                                    <el-select v-model="basicForm.mr5GuestEncryption" :disabled="wifiDisabled">
                                        <el-option label="WPA(AES)-PSK" value="WPA"></el-option>
                                        <el-option label="WPA2(AES)-PSK" value="WPA2"></el-option>
                                    </el-select>
                                </el-form-item>
                                <el-form-item label="Password" prop="mr5GuestPassPhrase">
                                    <el-input :type="airMap.g5GOpen?'text':'password'" v-model="basicForm.mr5GuestPassPhrase" class="with-append-input" :disabled="wifiDisabled">
                                        <span slot="append">
                                            <i class="el-icon margin-offset" 
                                                :class="{'background-color-white': true,'el-icon-operation-hide': !airMap.g5GOpen, 'el-icon-operation-view': airMap.g5GOpen}" 
                                                @click="airMap.g5GOpen = !airMap.g5GOpen">
                                            </i>
                                            Range: 8-64 bit strings in [a-za-z0-9_-]
                                        </span>
                                    </el-input>
                                </el-form-item>
                            </div>
                        </el-form>
                    </div>
                </el-collapse-item>

                <el-collapse-item name="Advance">
                    <template slot='title'>
                        <p style="display:inline-block;margin-left:40px;">
                            <span style="font-size:14px;font-weight:bold">Advance Settings</span>
                        </p>
                    </template>
                    <div class="rightContentCls" >
                        <el-form :model='advanceForm' ref="advanceForm" :rules="advanceRules" label-position="top">
                            <div class="paramsGroup">
                                <el-form-item prop='mrHideSSIDEnable' label="Broadcast SSID">
                                    <el-switch v-model="advanceForm.mrHideSSIDEnable" active-value="1" inactive-value="0" :disabled="wifiDisabled"></el-switch>
                                </el-form-item>
                                <el-form-item prop='mrAPIsolateEnable' label="APIsolated">
                                    <el-switch v-model="advanceForm.mrAPIsolateEnable" active-value="1" inactive-value="0" :disabled="wifiDisabled"></el-switch>
                                </el-form-item>
                                
                                <el-form-item label="Coverage" prop="mrTXpower">
                                    <el-select v-model="advanceForm.mrTXpower" :disabled="wifiDisabled">
                                        <el-option label="Short Coverage" value="1"></el-option>
                                        <el-option label="Medium Coverage" value="2"></el-option>
                                        <el-option label="Long Coverage" value="3"></el-option>
                                    </el-select>
                                </el-form-item>
                                <el-form-item label="Hwmode" prop="mrMode">
                                    <el-select v-model="advanceForm.mrMode" :disabled="wifiDisabled">
                                        <el-option label="WIFI5" value="WIFI5"></el-option>
                                        <el-option label="WIFI6" value="WIFI6"></el-option>
                                    </el-select>
                                </el-form-item>
                                <el-form-item label="2.4G Channel" prop="mr4Channel">
                                    <el-select v-model="advanceForm.mr4Channel" :disabled="wifiDisabled">
                                        <el-option v-for="item in channelList2G" :label="item" :value="item"></el-option>
                                    </el-select>
                                </el-form-item>
                                <el-form-item label="2.4G Bandwidth" prop="mr4BandWidth">
                                    <el-select v-model="advanceForm.mr4BandWidth" :disabled="wifiDisabled">
                                        <el-option label="20MHz" value="0"></el-option>
                                        <el-option label="20MHz/40MHz" value="1"></el-option>
                                    </el-select>
                                </el-form-item>
                                <el-form-item label="5G Channel" prop="mr5Channel">
                                    <el-select v-model="advanceForm.mr5Channel" :disabled="wifiDisabled">
                                        <el-option v-for="item in channelList5G" :label="item" :value="item"></el-option>
                                    </el-select>
                                </el-form-item>
                                <el-form-item label="5G Bandwidth" prop="mr5BandWidth">
                                    <el-select v-model="advanceForm.mr5BandWidth" :disabled="wifiDisabled">
                                        <el-option label="20MHz" value="0"></el-option>
                                        <el-option label="20MHz/40MHz" value="1"></el-option>
                                        <el-option label="20MHz/40MHz/80MHz" value="2"></el-option>
                                        <el-option label="20MHz/40MHz/80MHz/160MHz" value="3"></el-option>
                                    </el-select>
                                </el-form-item>
                            </div>
                        </el-form>
                    </div>
                </el-collapse-item>
            </el-collapse>
		</div>
		<div class='itemMainBoxFooter'>
			<el-button type="primary" @click="settingsSubmit"><%=rb.getString("QueDing")%></el-button>
			<el-button @click="closeSettings" ><%=rb.getString("QuXiao")%></el-button>
		</div>
	</div>
</div>

<script>
    new Vue({
	    el: '#cpeWifiConfigPage',
        data() {
            var vm = this,
                validPassPhraseM5G = (rule, value, cb) => {
                    var reg = /^[a-zA-Z0-9_\-]{8,64}$/;

                    if(value && reg.test(value)) {
                        cb();
                    }else {
                        cb(' ');
                    }
                },
                validPassPhraseM4G = (rule, value, cb) => {
                    var reg = /^[a-zA-Z0-9_\-]{8,64}$/,
                        needValid = vm.basicForm.mrCombineEnable === '0';

                    if(needValid) {
                        if(value && reg.test(value)) {
                            cb();
                        }else {
                            cb(' ');
                        }
                    }else {
                        cb();
                    }
                },
                validPassPhraseG5G = (rule, value, cb) => {
                    var reg = /^[a-zA-Z0-9_\-]{8,64}$/,
                        needValid = vm.basicForm.mrGuestEable == '1';

                    if(needValid) {
                        if(value && reg.test(value)) {
                            cb();
                        }else {
                            cb(' ');
                        }
                    }else {
                        cb();
                    }
                },
                validPassPhraseG4G = (rule, value, cb) => {
                    var reg = /^[a-zA-Z0-9_\-]{8,64}$/,
                        needValid = vm.basicForm.mrGuestEable == '1' && vm.basicForm.mrCombineEnable === '0';

                    if(needValid) {
                        if(value && reg.test(value)) {
                            cb();
                        }else {
                            cb(' ');
                        }
                    }else {
                        cb();
                    }
                };

            return {
                isRefresh: false,
                airMap: {
                    m4GOpen: false,
                    m5GOpen: false,
                    g4GOpen: false,
                    g5GOpen: false,
                },

                cpeCode: '',
                activeCollapse: 'Basic',
                basicForm: {
                    mrCombineEnable: '1',
                    mrGuestEable: '0',

                    mr4MasterSsid: '',
                    mr4MasterEncryption: '',
                    mr4MasterPassPhrase: '',
                    mr4GuestSsid: '',
                    mr4GuestEncryption: '',
                    mr4GuestPassPhrase: '',

                    mr5MasterSsid: '',
                    mr5MasterEncryption: '',
                    mr5MasterPassPhrase: '',
                    mr5GuestSsid: '',
                    mr5GuestEncryption: '',
                    mr5GuestPassPhrase: '',
                },
                rules: {
                    mr4MasterPassPhrase: [{validator: validPassPhraseM4G}],
                    mr4GuestPassPhrase: [{validator: validPassPhraseG4G}],
                    mr5MasterPassPhrase: [{validator: validPassPhraseM5G}],
                    mr5GuestPassPhrase: [{validator: validPassPhraseG5G}],
                },
                advanceForm: {
                    mrHideSSIDEnable: '0',
                    mrAPIsolateEnable: '0',
                    mrTXpower: '',
                    mrMode: '',
                    mr4Channel: '',
                    mr4BandWidth: '',
                    mr5Channel: '',
                    mr5BandWidth: '',
                },
                advanceRules: {

                },
                channelList2G: ['Auto','1','2','3','4','5','6','7','8','9','10','11','12','13'],
                channelList5G: ['Auto','36','40','44','48','52','56','60','64','149','153','157','161','165'],
            }
        },
        computed: {
            wifiDisabled() {
                var vm = this;

                return ['NotSupport','NotSync'].includes(vm.basicForm.mrCombineEnable);
            }
        },
        methods: {
            init(itemParam,code){
                var vm = this;
                vm.cpeCode = code;
                
                vm.getParamData(code);
            },
            getParamData(code) {
                var vm = this,
                    codes = [],
                    url = '${ctx}/cell/CPE/getSettingParams.action',
                    params = {
                        cpeCode: code
                    };
                
                axios.post(url, stringify(params)).then(function(res){
                    var data = res.data;
                    
                    ['mrCombineEnable','mrGuestEable','mr4MasterSsid','mr4MasterEncryption','mr4MasterPassPhrase','mr4GuestSsid','mr4GuestEncryption','mr4GuestPassPhrase',
                    'mr5MasterSsid','mr5MasterEncryption','mr5MasterPassPhrase','mr5GuestSsid','mr5GuestEncryption','mr5GuestPassPhrase'].map(function(item){
                        if(data[item] != undefined) vm.basicForm[item] = data[item];
                    });
                    
                    ['mrHideSSIDEnable','mrAPIsolateEnable','mrTXpower','mrMode','mr4Channel','mr4BandWidth','mr5Channel','mr5BandWidth'].map(function(item){
                        if(data[item] != undefined) vm.advanceForm[item] = data[item];
                    });

                    vm.$nextTick(function(){
                        initForm(vm.$refs.basicForm);
                        initForm(vm.$refs.advanceForm);
                    })
                });
            },
            refreshWiFi() {
                var vm = this,
                    params = {
                        cpeCode: vm.cpeCode
                    };
                
                    
                vm.isRefresh = true;
                axios.post('${ctx}/cell/CPE/setting/queryMrWifiInfo.action', stringify(params)).then(res=>{
                    var data = res.data;
                    
                    ['mrCombineEnable','mrGuestEable','mr4MasterSsid','mr4MasterEncryption','mr4MasterPassPhrase','mr4GuestSsid','mr4GuestEncryption','mr4GuestPassPhrase',
                    'mr5MasterSsid','mr5MasterEncryption','mr5MasterPassPhrase','mr5GuestSsid','mr5GuestEncryption','mr5GuestPassPhrase'].map(function(item){
                        vm.basicForm[item] = data[item];
                    });
                    
                    ['mrHideSSIDEnable','mrAPIsolateEnable','mrTXpower','mrMode','mr4Channel','mr4BandWidth','mr5Channel','mr5BandWidth'].map(function(item){
                        vm.advanceForm[item] = data[item];
                    });

                    vm.isRefresh = false;

                    vm.$nextTick(function(){
                        initForm(vm.$refs.basicForm);
                        initForm(vm.$refs.advanceForm);
                    })
                })
            },
            getFormChangedValues(form) {
                var params = {};

                if(form.isReinited){
                    form.fields.map(function(field){
                        if(Array.isArray(field.fieldValue)){
                            var vList = field.fieldValue.map(function(item){return item});
                            var oList = (field.reinitialValue||[]).map(function(item){return item});
                            var val = JSON.stringify(vList.sort());
                            var orVal = JSON.stringify(oList.sort());
                            if(val != orVal) {
                                params[field.prop] = val;
                            }
                        }else{
                            if(isNull(field.fieldValue) && isNull(field.reinitialValue)){
                                
                            }else if(field.fieldValue !== field.reinitialValue) {
                                params[field.prop] = field.fieldValue;
                            }
                        }
                    })
                }else{
                    form.fields.map(function(field){
                        if(Array.isArray(field.fieldValue)){
                            var vList = field.fieldValue.map(function(item){return item});
                            var oList = (field.initialValue||[]).map(function(item){return item});
                            var val = JSON.stringify(vList.sort()); 
                            var orVal = JSON.stringify(oList.sort());
                            if(val != orVal) {
                                params[field.prop] = val;
                            }
                        }else{
                            if(isNull(field.fieldValue) && isNull(field.initialValue)){
                                
                            }else if(field.fieldValue !== field.initialValue) {
                                params[field.prop] = field.fieldValue;
                            }
                        }
                    })
                }

                return params;

                function isNull(val){
                    if(val==undefined || val == null || val === "") return true;
                    else return false;
                }
            },
            settingsSubmit() {
                var vm = this,
                    hasChange = isFormChanged(vm.$refs.basicForm) || isFormChanged(vm.$refs.advanceForm);
                
                if(hasChange) {
                    var params = {
                            cpeCode: vm.cpeCode,
                            mrWifiChanged: 1,
                            mrCombineEnable: vm.basicForm.mrCombineEnable,
                        },
                        basicParams = vm.getFormChangedValues(vm.$refs.basicForm),
                        advanceParams = vm.getFormChangedValues(vm.$refs.advanceForm);

                    Object.assign(params, basicParams, advanceParams);
                    
                    vm.$refs.basicForm.validate(function(valid){
                        if(valid) {
                            $('#setting_main_cpe').addClass('loading');
                            axios.post('${ctx}/cell/CPE/setCpeParams.action', stringify(params)).then(function(res){
                                var data = res.data;

                                if(data["success"]){
                                    vm.$message({
                                        message: '<%=rb.getString("ChengGong")%>',
                                        type: 'success'
                                    });
                                    vm.closeSettings();
                                    $("#setting_main_cpe").removeClass("loading");
                                }else{
                                    vm.$message({
                                        message: data["message"],
                                        type: 'error'
                                    });
                                }
                                $('#setting_main_cpe').removeClass('loading');
                            });
                        }
                    })
                    
                }else {
                    vm.$message.error('No params change');
                }
            },
            closeSettings() {
                eventBus.$emit('close-cpe-setting');
            }
        },
        mounted() {
            eventBus.$off("cpe-data").$on("cpe-data",this.init);
        }
    })
</script>