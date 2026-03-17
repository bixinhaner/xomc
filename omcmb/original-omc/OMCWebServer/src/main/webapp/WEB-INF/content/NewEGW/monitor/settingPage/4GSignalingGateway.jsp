<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
#egw4GSignalingGatewayPage{
    width:100%;
    height:calc(100% - 10px);
    position: relative;
}
#egw4GSignalingGatewayPage .itemMainBoxCls{
	border:1px solid #d5dcec;
	border-radius:10px;
	margin:8px;
	background:#fff;
	height:100%;
}
#egw4GSignalingGatewayPage .itemMainBoxCls .el-tab-pane{
	position: relative;
}
#egw4GSignalingGatewayPage .itemMainBoxTitle {
	height:36px;
    line-height: 36px;
	font-size:14px;
	font-weight:bold;
    position: relative;
}
#egw4GSignalingGatewayPage .egwTabPaneItemBoxCls{
	flex: 1;
}
#egw4GSignalingGatewayPage .egwTabPaneTableBoxCls{
    height:calc(100% - 120px);
    overflow: hidden;
    box-sizing: border-box;
    border:1px solid #d5dcec;
    border-radius: 4px;
    margin: 0px 50px 0px 20px;
}
#egw4GSignalingGatewayPage .toolbarBoxCls{
    padding: 10px 0px;
}
#egw4GSignalingGatewayPage .egwTabPaneContent{
    margin: 25px 0px 0px 40px;
    display: flex;
    flex-direction: column;
    height:calc(100% - 75px);
    overflow: auto;
}
#egw4GSignalingGatewayPage .egwTabPaneContent .egwTabPaneItemCls{
    margin-bottom: 20px;
    font-size: 14px;
}
#egw4GSignalingGatewayPage .itemMainBoxFooter{
    display: flex;
    align-items: center;
    border-top : 1px solid #E9E9E9;
	height:48px;
    box-sizing: border-box;
    width: calc(100% - 0px);
    position: absolute;
    bottom: 0px;
    background: #FFFFFF;
}
.gnbConfigAddDialog .el-form{
    display: flex;
    flex-wrap: wrap;
    justify-content: space-between;
    margin-right: 40px;
}
.gnbConfigAddDialog .el-form-item{
    display: inline-block;
    width: 48%;
}
.gnbConfigAddDialog .el-form-item__error{
    padding-top: 0px;
    top:30px!important;
}
#egw4GSignalingGatewayPage .egwTabPaneContent .el-input__suffix{
    height: 26px;
    display: flex;
    align-items: center;
}
</style>

<div id="egw4GSignalingGatewayPage">
    <div class="newIconBoxCls-bt" style="right:20px;top:5px;" @click="syncSubmit" tip='<%=rb.getString("TongBu")%>'>
        <span class="el-icon el-icon-circle-refresh"></span>
    </div>	
	<div class="itemMainBoxCls">
		<el-tabs class="fit newTabs" v-model="activeName" style='height:100%' @tab-click="tabClick">
            <el-tab-pane label="Basic configuration" name="Basic">
                <div class="egwTabPaneContent">
                     <div class="itemMainBoxTitle">
                        <span class="el-icon el-icon-splitGroup"></span>
                        <%=rb.getString("JiChuPeiZhi")%>
                    </div>
                    <el-form ref="enbSettingBasicForm" :model="enbSettingBasicForm" :rules="enbBasicFormRules" label-position="top"  :hide-required-asterisk='true'>
                        <div class="allowMoreInputBoxCls">
                            <div class="allowMoreInputHeadCls">
                                <span class="allowMoreInputTitleCls">PLMN</span>
                                <span class="allowMoreInputTipsCls">（Add 8 at most）</span>
                            </div>
                            <div class="allowMoreInputContentCls">
                                <div class="allowMoreInputFieldCls">
                                    <el-input v-model="enbSettingBasicForm.plmn"></el-input>
                                    <div class="allowMoreInputAddBtnCls" @click="addEnbPLMNs" v-show="enbPlmnList.length < 8 ">
                                        <span class="el-icon el-icon-plus"></span>
                                        <span>Add</span>
                                    </div>
                                </div>
                                <div class="allowMoreInputParamsCls">
                                    <div v-for="item in enbPlmnList" class="allowMoreInputParamsItemCls">
                                        <span>{{item.Hplmn}}</span>
                                        <span class="el-icon el-icon-close" style="margin-left:5px;" @click="enbPlmnListDel(item)"></span>
                                    </div>
                                </div>
                            </div>
                            <div class="allowMoreInputFootCls">
                                <p class="inputErrorBoxCls">{{enbPlmnErrorMessage}}</p>
                            </div>
                        </div>
                        <div class="allowMoreInputBoxCls">
                            <div class="allowMoreInputHeadCls">
                                <span class="allowMoreInputTitleCls"><%=rb.getString("eGWIP")%></span>
                                <span class="allowMoreInputTipsCls">（Add 4 at most,Support IPv4 or IPv6）</span>
                            </div>
                            <div class="allowMoreInputContentCls">
                                <div class="allowMoreInputFieldCls">
                                    <el-input v-model="enbSettingBasicForm.egwIp"></el-input>
                                    <div class="allowMoreInputAddBtnCls" @click="addEgwIp" v-show="enbEgwIpList.length < 4">
                                        <span class="el-icon el-icon-plus"></span>
                                        <span>Add</span>
                                    </div>
                                </div>
                                <div class="allowMoreInputParamsCls">
                                    <div v-for="item in enbEgwIpList" class="allowMoreInputParamsItemCls">
                                        <span>{{item.Ip}}</span>
                                        <span class="el-icon el-icon-close" style="margin-left:5px;" @click="egwIpListDel(item)"></span>
                                    </div>
                                </div>
                            </div>
                            <div class="allowMoreInputFootCls">
                                <p class="inputErrorBoxCls">{{ipErrorMessage}}</p>
                            </div>
                        </div>
                        <el-form-item prop="egwPort" label='<%=rb.getString("EGWDuanKou")%>' label-width="160px" style="margin:10px 0px 20px 25px;">
                            <el-input style='width:200px;' v-model="enbSettingBasicForm.egwPort" :disabled="true"></el-input>
                        </el-form-item>
                        <div class="allowMoreInputBoxCls">
                            <div class="allowMoreInputHeadCls">
                                <span class="allowMoreInputTitleCls">UpLink S1-U IP</span>
                                <span class="allowMoreInputTipsCls">（Add 8 at most,Support IPv4 or IPv6）</span>
                            </div>
                            <div class="allowMoreInputContentCls">
                                <div class="allowMoreInputFieldCls">
                                    <el-input v-model="enbSettingBasicForm.upLinkIp"></el-input>
                                    <div class="allowMoreInputAddBtnCls" @click="addEnbUpLinkIp" v-show="enbUpLinkIpList.length < 8 ">
                                        <span class="el-icon el-icon-plus"></span>
                                        <span>Add</span>
                                    </div>
                                </div>
                                <div class="allowMoreInputParamsCls">
                                    <div v-for="item in enbUpLinkIpList" class="allowMoreInputParamsItemCls">
                                        <span>{{item.UplinkAddr}}</span>
                                        <span class="el-icon el-icon-close" style="margin-left:5px;" @click="enbUpLinkIpListDel(item)"></span>
                                    </div>
                                </div>
                            </div>
                            <div class="allowMoreInputFootCls">
                                <p class="inputErrorBoxCls">{{enbUpLinkIpErrorMessage}}</p>
                            </div>
                        </div>
                        <div class="allowMoreInputBoxCls">
                            <div class="allowMoreInputHeadCls">
                                <span class="allowMoreInputTitleCls">DownLink S1-U IP</span>
                                <span class="allowMoreInputTipsCls">（Add 8 at most,Support IPv4 or IPv6）</span>
                            </div>
                            <div class="allowMoreInputContentCls">
                                <div class="allowMoreInputFieldCls">
                                    <el-input v-model="enbSettingBasicForm.downLinkIp"></el-input>
                                    <div class="allowMoreInputAddBtnCls" @click="addEnbDownLinkIp" v-show="enbDownLinkIpList.length < 8 ">
                                        <span class="el-icon el-icon-plus"></span>
                                        <span>Add</span>
                                    </div>
                                </div>
                                <div class="allowMoreInputParamsCls">
                                    <div v-for="item in enbDownLinkIpList" class="allowMoreInputParamsItemCls">
                                        <span>{{item.DownlinkAddr}}</span>
                                        <span class="el-icon el-icon-close" style="margin-left:5px;" @click="enbDownLinkIpListDel(item)"></span>
                                    </div>
                                </div>
                            </div>
                            <div class="allowMoreInputFootCls">
                                <p class="inputErrorBoxCls">{{enbDownLinkIpErrorMessage}}</p>
                            </div>
                        </div>
                        
                    </el-form>
                </div>
                <div class='itemMainBoxFooter'>
                    <el-button type="primary" @click="submitBasicAndEnbList" style="margin-left:20px;"><%=rb.getString("QueDing")%></el-button>
                    <el-button @click="closeSettingPage"><%=rb.getString("QuXiao")%></el-button>
                </div>
            </el-tab-pane>
            <el-tab-pane label="Link Configuration" name="enbConfig">
                <div class="egwTabPaneContent">
                    <div class="itemMainBoxTitle">
                        <span class="el-icon el-icon-splitGroup"></span>
                        eNB ID List
                        <span class="el-icon el-icon-circle-add" @click="linkAdd('enb')" style="position:absolute;right:55px;"></span>
                    </div>
                    <div class="egwTabPaneTableBoxCls">
                        <el-ctable 
                            ref="enbIdMmeListTable" 
                            :rownumber="true" 
                            id="enbIdMmeListTable" 
                            :data="enbIdMmeListTableData" 
                            height="100%"
                            :front-pagination="true"
                            :pagination="true"
                         >
                            <el-table-column label='<%=rb.getString("CaoZuo")%>' min-width="100">
                                <template slot-scope="scope">
                                    <span class="el-icon el-icon-operation-edit" @click="editEnbIdMme(scope.row,event)"></span>
                                    <span class="el-icon el-icon-operation-delete" @click="delEnbIdMme(scope.row,event)" style="margin-left:20px;"></span>
                                </template>
                            </el-table-column>
                            <el-table-column label='eNodeB ID' min-width="120" prop="enodebId" show-overflow-tooltip></el-table-column>
                            <el-table-column label='Link Num' min-width="120" prop="linkNum" show-overflow-tooltip></el-table-column>
                            <el-table-column label='PLMN' min-width="120" prop="Hplmn" show-overflow-tooltip></el-table-column>
                            <el-table-column label='TAC' min-width="120" prop="Tac" show-overflow-tooltip></el-table-column>
                        </el-ctable>
                    </div>
                </div>
                <div class='itemMainBoxFooter'>
                    <el-button type="primary" @click="submitBasicAndEnbList" style="margin-left:20px;"><%=rb.getString("QueDing")%></el-button>
                    <el-button @click="closeSettingPage"><%=rb.getString("QuXiao")%></el-button>
                </div>
            </el-tab-pane>
		</el-tabs>
	</div>
    <el-slide ref="egwSettingLinkSlide" id="egwSettingLinkSlide" :url='settingLinkSlideUrl' :title="settingLinkSlideTitle" :footer="settingLinkSlideFooter" :header="settingLinkSlideHeader" 
        :position="settingLinkSlidePosition" :height="settingLinkSlideHeight"  :width='settingLinkSlideWidth' @ok="settingLinkSubmit"  @cancel="closeSettingLinkSlide" :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
    </el-slide>
</div>

<script>
var regIp = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/,
    regKey = /^[A-Fa-f0-9]{32}$/,
    regNumber = /^[0-9]{15}$/;
var egw4GSignalingGatewayPage = new Vue({
	el: '#egw4GSignalingGatewayPage', 
	data() {
		var vm = this;
        var validateIPorMask = function(rule,value,callback) {
                var regIpOrMask = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\/([0-9]|[12][0-9]|3[012])$/,
                    regIPV6OrMask = /^([\da-fA-F]{1,4}:){6}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^::([\da-fA-F]{1,4}:){0,4}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:):([\da-fA-F]{1,4}:){0,3}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:){2}:([\da-fA-F]{1,4}:){0,2}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:){3}:([\da-fA-F]{1,4}:){0,1}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:){4}:((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:){7}[\da-fA-F]{1,4}\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^:((:[\da-fA-F]{1,4}){1,6}|:)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^[\da-fA-F]{1,4}:((:[\da-fA-F]{1,4}){1,5}|:)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:){2}((:[\da-fA-F]{1,4}){1,4}|:)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:){3}((:[\da-fA-F]{1,4}){1,3}|:)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:){4}((:[\da-fA-F]{1,4}){1,2}|:)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:){5}:([\da-fA-F]{1,4})?\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:){6}:\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$/;
                if( value === '' || value === null || value === undefined) {
                    callback(new Error('<%=rb.getString("IPMaskGeShiTiShi")%>'))
                }else {
                    if(regIpOrMask.test(value) || regIPV6OrMask.test(value)){
                        callback();
                    }else{
                        callback(new Error('<%=rb.getString("IPMaskGeShiTiShi")%>'))
                    }
                    
                }
            },
            validateRouteIP = function(rule,value,callback) {
                
                if( value === '' || value === null || value === undefined) {
                    callback('Example:192.168.9.0');
                }else {
                    if(regIp.test(value) || vm.isIPv6(value)){
                        callback();
                    }else{
                        callback('Example:192.168.9.0');
                    }
                    
                }
            },
            validatePort = (rule,value,callback) => {
				if(value === '' || value === null || value === undefined){
                    callback(new Error('<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%> 0~65535 <%=rb.getString("ZhengXing")%>'))
                }else{
                    if(vm.isInteger(value)&& parseInt(value)>=0 && parseInt(value)<=65535){
                        callback();
                    }else{
                        callback(new Error('<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%> 0~65535 <%=rb.getString("ZhengXing")%>'))
                    }
                }
			};
		return {
            egwCode:'',
            egwSn:'',
            activeName:'Basic',
            enbSettingBasicForm:{
                plmn:'',
                egwIp:'',
                egwPort:'36412',
                upLinkIp:'',
                downLinkIp:'',
            },
            enbBasicFormRules:{},

            enbPlmnList:[],
            enbEgwIpList:[],
            enbUpLinkIpList:[],
            enbDownLinkIpList:[],
            enbIdMmeListTableData:[],
            delEnbIdMmeList:[],
            delEnbPlmnList:[],
            delIpList:[],
            delEnbUplinkAddrList:[],
            delEnbDownlinkAddrList:[],
            enbPlmnErrorMessage:'',
            ipErrorMessage:'',
            enbUpLinkIpErrorMessage:'',
            enbDownLinkIpErrorMessage:'',
            oldEnbDataList:{
                basicInfo:{},
                enbListInfo:[]
            },

            settingLinkSlideUrl:'',
			settingLinkSlideTitle:'',
			settingLinkSlideHeader:'',
			settingLinkSlideFooter:'',
			settingLinkSlidePosition:'',
			settingLinkSlideHeight:'',
			settingLinkSlideWidth:'',

		};
	},
	computed: {},
	methods: {
        // 初始化
		init(row,code,sn,status){
		    var vm =this;
			vm.egwCode = code;
            vm.egwSn = sn;
            vm.init4GParams();
		},
        init4GParams(){
            var vm = this,
                code = vm.egwCode,
                params={
                    egwCode:code
                };
            axios.post('${ctx}/egw/config/getBasicAndENBListConfig.action',stringify(params)).then(function(response){
                var data = response.data;
                if(data){
                    var basicInfo = data.basicInfo,
                        enbListInfo = data.enbListInfo;
                    
                    vm.enbIdMmeListTableData = enbListInfo ? enbListInfo : [];
                    if(basicInfo.HplmnList){
                        vm.enbPlmnList = basicInfo.HplmnList;
                    }
                    if(basicInfo.IpList){
                        vm.enbEgwIpList = basicInfo.IpList;
                    }
                    if(basicInfo.UplinkAddrList){
                        vm.enbUpLinkIpList = basicInfo.UplinkAddrList;
                    }
                    if(basicInfo.DownlinkAddrList){
                        vm.enbDownLinkIpList = basicInfo.DownlinkAddrList;
                    }
                    var initData = {
                        HplmnList:basicInfo.HplmnList||[],
                        IpList:basicInfo.IpList||[],
                        UplinkAddrList:basicInfo.UplinkAddrList||[],
                        DownlinkAddrList:basicInfo.DownlinkAddrList||[],
                    }
                    vm.oldEnbDataList.basicInfo = JSON.parse(JSON.stringify(initData));
                    vm.oldEnbDataList.enbListInfo = JSON.parse(JSON.stringify(enbListInfo));
                }
            }).catch(function(error){});
        },
        // Tab 切换
		tabClick(val){
            var vm = this,
                str = Math.random().toString();
        },
        //enb  PLMN 添加事件
        addEnbPLMNs(){
            var vm = this,
                val = vm.enbSettingBasicForm.plmn,
                reg = /^\d{5,6}$/,
                params={
                    Hplmn:vm.enbSettingBasicForm.plmn
                },
                result = vm.enbPlmnList.some(item=>item.Hplmn == val);
            
            if(val){
                if(reg.test(val)) {
                    if(result){
                        vm.enbPlmnErrorMessage = '<%=rb.getString("YiCunZai")%>';
                    }else{
                        vm.enbPlmnList.push(params);
                        vm.enbPlmnErrorMessage = '';
                        vm.enbSettingBasicForm.plmn = '';
                    }
                }else{
                    vm.enbPlmnErrorMessage = '<%=rb.getString("PLMNFanWei")%>';
                }
            }
            
        },
        // enb  PLMN 删除事件
        enbPlmnListDel(row,type){
            var vm = this;

            if(row.configIndex){
                vm.delEnbPlmnList.push(row.configIndex);
            }
            vm.enbPlmnList = vm.enbPlmnList.filter((items)=>{
                return items.Hplmn != row.Hplmn
            })
        },
        // EGWIP 添加事件
        addEgwIp(){
            var vm = this,
                val = vm.enbSettingBasicForm.egwIp,
                params={
                    Ip:vm.enbSettingBasicForm.egwIp
                };
            if(val){
                if(vm.isValidIP(val) || vm.isIPv6(val)) {
                    var result = vm.enbEgwIpList.some(item=>item.Ip == val);
                    if(result){
                        vm.ipErrorMessage = '<%=rb.getString("YiCunZai")%>';
                    }else{
                        vm.enbEgwIpList.push(params);
                        vm.enbSettingBasicForm.egwIp = '';
                        vm.ipErrorMessage = '';
                    }
                }else {
                    vm.ipErrorMessage = 'Support configuration of IPV4 or IPV6';
                }
            }
        },
        // EGWIP 删除事件
        egwIpListDel(row){
            var vm = this;

            if(row.configIndex){
                vm.delIpList.push(row.configIndex);
            }
            vm.enbEgwIpList = vm.enbEgwIpList.filter((items)=>{
                return items.Ip != row.Ip
            })
        },
        //enb  UpLink IP添加事件
        addEnbUpLinkIp(){
            var vm = this,
                val = vm.enbSettingBasicForm.upLinkIp,
                params={
                    UplinkAddr:vm.enbSettingBasicForm.upLinkIp
                };
            if(val){
                if(vm.isValidIP(val) || vm.isIPv6(val)) {
                    var result = vm.enbUpLinkIpList.some(item=>item.UplinkAddr == val);
                    if(result){
                        vm.enbUpLinkIpErrorMessage = '<%=rb.getString("YiCunZai")%>';
                    }else{
                        vm.enbUpLinkIpList.push(params);
                        vm.enbSettingBasicForm.upLinkIp = '';
                        vm.enbUpLinkIpErrorMessage = '';
                    }
                }else {
                    vm.enbUpLinkIpErrorMessage = 'Support configuration of IPV4 or IPV6';
                }
            }
            
        },
        // enb UpLink IP 删除事件
        enbUpLinkIpListDel(row){
            var vm = this;
            if(row.configIndex){
                vm.delEnbUplinkAddrList.push(row.configIndex);
            }
            vm.enbUpLinkIpList = vm.enbUpLinkIpList.filter((items)=>{
                return items.UplinkAddr != row.UplinkAddr
            })
        },
        // enb downLink IP添加事件
        addEnbDownLinkIp(){
            var vm = this,
                val = vm.enbSettingBasicForm.downLinkIp,
                params={
                    DownlinkAddr:vm.enbSettingBasicForm.downLinkIp
                };

            if(val){
                if(vm.isValidIP(val) || vm.isIPv6(val)) {
                    var result = vm.enbDownLinkIpList.some(item=>item.DownlinkAddr == val);
                    if(result){
                        vm.enbDownLinkIpErrorMessage = '<%=rb.getString("YiCunZai")%>';
                    }else{
                        vm.enbDownLinkIpList.push(params);
                        vm.enbSettingBasicForm.downLinkIp = '';
                        vm.enbDownLinkIpErrorMessage = '';
                    }
                }else {
                    vm.enbDownLinkIpErrorMessage = 'Support configuration of IPV4 or IPV6';
                }
            }
        },
        //  downLink IP 删除事件
        enbDownLinkIpListDel(row){
            var vm = this;
            if(row.configIndex){
                vm.delEnbDownlinkAddrList.push(row.configIndex);
            }
            vm.enbDownLinkIpList = vm.enbDownLinkIpList.filter((items)=>{
                return items.DownlinkAddr != row.DownlinkAddr
            })
        },
        // enb 基本配置 基站配置 提交
        submitBasicAndEnbList(){
            var vm = this;

            if(vm.activeName == 'Basic'){
                var errShow = false;
                if(vm.enbPlmnList.length == 0){
                    vm.enbPlmnErrorMessage = '<%=rb.getString("ZhiShaoTianJiaYiGe")%>';
                    errShow = true;
                }
                if(vm.enbEgwIpList.length == 0){
                    vm.ipErrorMessage = '<%=rb.getString("ZhiShaoTianJiaYiGe")%>';
                    errShow = true;
                }
                if(vm.enbUpLinkIpList.length == 0){
                    vm.enbUpLinkIpErrorMessage = '<%=rb.getString("ZhiShaoTianJiaYiGe")%>';
                    errShow = true;
                }
                if(vm.enbDownLinkIpList.length == 0){
                    vm.enbDownLinkIpErrorMessage = '<%=rb.getString("ZhiShaoTianJiaYiGe")%>';
                    errShow = true;
                }
                if(errShow) return
            }
            
            let params = {},
                enbListInfo=[],
                addHplmnList = [],
                addIpList = [],
                addUplinkAddrList = [],
                addDownlinkAddrList = [],
                basicInfo={};
            if(vm.activeName == 'Basic'){
                vm.enbPlmnList.map((item,index)=>{
                    if(!item.configIndex){
                        addHplmnList.push(item)
                    }
                })
                vm.enbEgwIpList.map((item,index)=>{
                    if(!item.configIndex){
                        addIpList.push(item)
                    }
                })
                vm.enbUpLinkIpList.map((item,index)=>{
                    if(!item.configIndex){
                        addUplinkAddrList.push(item)
                    }
                })
                vm.enbDownLinkIpList.map((item,index)=>{
                    if(!item.configIndex){
                        addDownlinkAddrList.push(item)
                    }
                })
            }
            
            basicInfo.HplmnList = addHplmnList;
            basicInfo.IpList = addIpList;
            basicInfo.UplinkAddrList = addUplinkAddrList;
            basicInfo.DownlinkAddrList = addDownlinkAddrList;
            params.egwCode = vm.egwCode;
            
            params.basicInfo = JSON.stringify(basicInfo);

            if(vm.activeName == 'Basic'){
                params.delPlmnList = vm.delEnbPlmnList.join(',');
                params.delIpList = vm.delIpList.join(',');
                params.delUplinkAddrList = vm.delEnbUplinkAddrList.join(',');
                params.delDownlinkAddrList = vm.delEnbDownlinkAddrList.join(',');
            }else{
                vm.enbIdMmeListTableData.map((item,index)=>{
                    if(item.isEdit && item.isEdit == 'true'){
                        enbListInfo.push(item)
                    }
                })
                enbListInfo.map((item)=>{
                    if(item.enbLinkListInfo && item.enbLinkListInfo.length >0){
                        item.enbLinkListInfo = item.enbLinkListInfo.filter((items)=>{
                            return !items.configIndex
                        })
                    }
                })
                params.delEnbList = vm.delEnbIdMmeList.join(',');
            }
            params.enbListInfo = JSON.stringify(enbListInfo);
            axios.post("${ctx}/egw/config/setBasicAndENBListConfig.action",stringify(params)).then(function(response){
                var data = response.data;
                if(data["success"]){
                    vm.$message({
                        message:'<%=rb.getString("ChengGong")%>',
                        type:'success',
                    })
                    vm.closeSettingPage();
                }else{
                    vm.$message.error('<%=rb.getString("ShiBai")%>') 
                }
            })
        },
        // 关闭配置页面
        closeSettingPage(){
            var vm = this;
            egwMonitor.$refs.egwSettingPage.hide();
        },
        // 新增链路
        linkAdd(type){
            var vm = this;

            vm.settingLinkSlideHeader = true;
            vm.settingLinkSlideUrl = '${ctx}/egw/pageForward/goEGWSettingsEnbLink.action';
            vm.settingLinkSlideFooter = true;
            vm.settingLinkSlidePosition = 'top';
            vm.settingLinkSlideHeight = '100%';
            vm.settingLinkSlideWidth = '100%';
            vm.settingLinkSlideTitle = 'add';
            vm.$refs.egwSettingLinkSlide.showSlide(()=>{
                eventBus.$emit("link-init",'add','','enb');
            });
        },
        // 修改 链路总配置
        editEnbIdMme(row,event){
            var vm = this;

            vm.settingLinkSlideHeader = true;
            vm.settingLinkSlideUrl = '${ctx}/egw/pageForward/goEGWSettingsEnbLink.action';
            vm.settingLinkSlideFooter = true;
            vm.settingLinkSlidePosition = 'top';
            vm.settingLinkSlideHeight = '100%';
            vm.settingLinkSlideWidth = '100%';
            vm.settingLinkSlideTitle = 'edit';
            vm.$refs.egwSettingLinkSlide.showSlide(()=>{
                eventBus.$emit("link-init",'edit',row,'enb');
            });
        },
        // 删除 enb 链路总配置
        delEnbIdMme(row,event){
            var vm = this;
            if(row.configIndex){
                vm.delEnbIdMmeList.push(row.configIndex);
            }
            vm.enbIdMmeListTableData = vm.enbIdMmeListTableData.filter((items)=>{
                return (items.enodebId +''+ items.Hplmn) != (row.enodebId +''+ row.Hplmn)
            })
        },
        settingLinkSubmit(){
			var vm = this;
			eventBus.$emit("egw-settingLink-ok");
		},
        // 链路设置保存 更改enbList表格信息
        editEnbList(data,type){
            var vm = this,
                editDataList = data,
                isExist = false;
            vm.enbIdMmeListTableData.forEach((items,index,array)=>{
                if((items.enodebId +''+ items.Hplmn) == (data.enodebId +''+ data.Hplmn)){
                    isExist = true;
                    array[index].linkNum = data.linkNum;
                    array[index].Hplmn = data.Hplmn;
                    array[index].Tac = data.Tac;
                    array[index].enbLinkListInfo = data.enbLinkListInfo;
                    array[index].isEdit = data.isEdit;
                    if(data.delEnbLinkList){
                        array[index].delEnbLinkList = data.delEnbLinkList;
                    }else{
                        array[index].delEnbLinkList = '';
                    }
                }
            });
            if(!isExist){
                vm.enbIdMmeListTableData.unshift(data);
            }
        },
		// 关闭链路配置置页面
		closeSettingLinkSlide(){
			var vm = this;
			vm.$refs.egwSettingLinkSlide.hide();
		},
        //校验IP
        isValidIP(ip){
            var reg =  /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/     
            return reg.test(ip);     
        },
        //Ipv6校验 
        isIPv6(str){ 
            var reg = /^([\da-fA-F]{1,4}:){6}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^::([\da-fA-F]{1,4}:){0,4}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:):([\da-fA-F]{1,4}:){0,3}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){2}:([\da-fA-F]{1,4}:){0,2}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){3}:([\da-fA-F]{1,4}:){0,1}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){4}:((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){7}[\da-fA-F]{1,4}$|^:((:[\da-fA-F]{1,4}){1,6}|:)$|^[\da-fA-F]{1,4}:((:[\da-fA-F]{1,4}){1,5}|:)$|^([\da-fA-F]{1,4}:){2}((:[\da-fA-F]{1,4}){1,4}|:)$|^([\da-fA-F]{1,4}:){3}((:[\da-fA-F]{1,4}){1,3}|:)$|^([\da-fA-F]{1,4}:){4}((:[\da-fA-F]{1,4}){1,2}|:)$|^([\da-fA-F]{1,4}:){5}:([\da-fA-F]{1,4})?$|^([\da-fA-F]{1,4}:){6}:$/
            return reg.test(str);
        },
         // 验证输入的是否是整数
        isInteger(str) {
            if(str.length==0){
                return false;
            }
            var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;
            if(!reg.test(str)){
                return false;
            }
            return true;  
        },
        // 同步
        syncSubmit(){
            var vm = this,
                urls='${ctx}/egw/config/refreshConfig.action',
                params={
                    egwCode: vm.egwCode,
                    refreshType: 'SIGNALINGGATEWAY4G'
                };
            axios.post(urls,stringify(params)).then(res=>{
                var data = res.data;
                if(data["success"]){
                    vm.$message({
                        message: '<%=rb.getString("MingLingYiXiaFa")%>',
                        type:'success',
                    });
                }else{
                    vm.$message.error(data["message"])
                }
            })
        },
	},
	mounted() {
		eventBus.$off("egw-data").$on("egw-data",this.init);
        eventBus.$off('edit-enbList').$on('edit-enbList',this.editEnbList);
        eventBus.$off('close-linkSetting').$on('close-linkSetting',this.closeSettingLinkSlide);
	}
});

</script>
