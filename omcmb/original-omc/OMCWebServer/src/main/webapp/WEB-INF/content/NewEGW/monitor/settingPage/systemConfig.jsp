<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
#egwSystemConfigPage{
    width:100%;
    height:calc(100% - 10px);
    position: relative;
}
#egwSystemConfigPage .itemMainBoxCls{
	border:1px solid #d5dcec;
	border-radius:10px;
	margin:8px;
	background:#fff;
	height:100%;
}
#egwSystemConfigPage .itemMainBoxCls .el-tab-pane{
	position: relative;
}
#egwSystemConfigPage .itemMainBoxTitle {
	height:36px;
    line-height: 36px;
	font-size:14px;
	font-weight:bold;
    position: relative;
}
#egwSystemConfigPage .egwTabPaneItemBoxCls{
	flex: 1;
}
#egwSystemConfigPage .egwTabPaneTableBoxCls{
    height:calc(100% - 60px);
    overflow: hidden;
    box-sizing: border-box;
    border:1px solid #d5dcec;
    border-radius: 4px;
    margin: 0px 50px 0px 20px;
}
#egwSystemConfigPage .toolbarBoxCls{
    padding: 10px 0px;
}
#egwSystemConfigPage .egwTabPaneContent{
    margin: 25px 0px 0px 40px;
    display: flex;
    flex-direction: column;
    height: calc(100% - 60px);
    overflow: auto;
}
#egwSystemConfigPage .egwTabPaneContent .egwTabPaneItemCls{
    margin-bottom: 20px;
    font-size: 14px;
}
#egwSystemConfigPage .itemMainBoxFooter{
    display: flex;
    align-items: center;
    background: #fff;
    border-top : 1px solid #E9E9E9;
	height:48px;
    box-sizing: border-box;
    width: calc(100% - 0px);
    position: absolute;
    bottom: 0px;
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
</style>

<div id="egwSystemConfigPage">
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
                    <el-form ref="commonConfigForm" :model="commonConfigForm" label-position="top"  :hide-required-asterisk='true'>
                        <div class="egwTabPaneItemCls">
                            <el-form-item prop="egwName" label="<%=rb.getString("EGWMingCheng")%>"  label-width="160px" style="margin:10px 0px 10px 25px;">
                                <el-input style='width:200px;' v-model="commonConfigForm.egwName"></el-input>
                            </el-form-item>
                        </div>
                        <div class="egwTabPaneItemCls">
                            <el-form-item prop="egwDescription" label="<%=rb.getString("MiaoShu")%>"  label-width="160px" style="margin:20px 0px 0px 25px;">
                                <el-input type="textarea" resize="true" :rows="3" maxlength="500" v-model="commonConfigForm.egwDescription" style="width: 560px;"></el-input>
                            </el-form-item>
                        </div>
                    </el-form>
                </div>
                <div class='itemMainBoxFooter'>
                    <el-button type="primary" @click="submitCommonConfig" style="margin-left:20px;"><%=rb.getString("QueDing")%></el-button>
                    <el-button @click="closeLinkSetting"><%=rb.getString("QuXiao")%></el-button>
                </div>
            </el-tab-pane>
            <el-tab-pane label="Linux" name="linux">
                <div class="egwTabPaneContent">
                    <div class="egwTabPaneItemBoxCls">
                        <div class="itemMainBoxTitle">
                            <span class="el-icon el-icon-splitGroup"></span>
                            Set Route
                            <span class="el-icon el-icon-circle-add" style="position:absolute;right:55px;" @click="routeAdd('linux')"></span>
                        </div>
                        <div class="egwTabPaneTableBoxCls">
                            <el-ctable 
                                ref="linuxRouteListTable"
                                :rownumber="true" 
                                id="linuxRouteListTable" 
                                :url="linuxRouteListTableUrl"
                                :time="6"
                                height="100%" 
                                pagination="true"
                             >
                                <el-table-column label='<%=rb.getString("CaoZuo")%>' width="100" fixed>
                                    <template slot-scope="scope">
                                        <span class="el-icon el-icon-circle-warning yellowIcon" title="Added not Send" v-if="scope.row.dataStatus == '2'"></span>
                                        <span class="el-icon el-icon-circle-warning redIcon" title="Send Failure" v-if="scope.row.dataStatus == '4'"></span>
                                        <span class="el-icon el-icon-operation-delete" @click="delRoute(scope.row,'linux',event)" style="margin-left:5px;"></span>
                                    </template>
                                </el-table-column>
                                <el-table-column label='TargetIP' min-width="120" prop="TargetIp" show-overflow-tooltip></el-table-column>
                                <el-table-column label='NextHopIP' min-width="120" prop="NextHop" show-overflow-tooltip></el-table-column>
                                <el-table-column label='Device' min-width="120" prop="Device" show-overflow-tooltip></el-table-column>
                            </el-ctable>
                        </div>
                    </div>
                    <div class="egwTabPaneItemBoxCls">
                        <div class="itemMainBoxTitle">
                            <span class="el-icon el-icon-splitGroup"></span>
                            ACL Configuration
                            <span class="el-icon el-icon-circle-add" style="position:absolute;right:55px;" @click="aclAdd('linux')"></span>
                        </div>
                        <div class="egwTabPaneTableBoxCls">
                            <el-ctable 
                                ref="linuxAclListTable" 
                                :rownumber="false" 
                                id="linuxAclListTable" 
                                :url="linuxAclListTableUrl"
                                :time="6"
                                height="100%"
                                pagination="true"
                             >
                                <el-table-column label='<%=rb.getString("CaoZuo")%>' width="100" fixed>
                                    <template slot-scope="scope">
                                        <span class="el-icon el-icon-circle-warning yellowIcon" title="Added not Send" v-if="scope.row.dataStatus == '2'"></span>
                                        <span class="el-icon el-icon-circle-warning redIcon" title="Send Failure" v-if="scope.row.dataStatus == '4'"></span>
                                        <span class="el-icon el-icon-operation-delete" @click="delAcl(scope.row,'linux',event)" style="margin-left:5px;"></span>
                                    </template>
                                </el-table-column>
                                <el-table-column label='<%=rb.getString("WangKaMingCheng")%>' min-width="150" prop="NetCardName"></el-table-column>
                                <el-table-column label='<%=rb.getString("ShuJuFangXiang")%>' min-width="120" prop="direction" show-overflow-tooltip></el-table-column>
                                <el-table-column label='<%=rb.getString("YuanIP")%>' min-width="100" prop="SourceIp"></el-table-column>
                                <el-table-column label='Target IP' min-width="120" prop="TargetIp" show-overflow-tooltip></el-table-column>
                                <el-table-column label='<%=rb.getString("XieYiLeiXing")%>' min-width="120" prop="ProtocolCode"></el-table-column>
                                <el-table-column label='<%=rb.getString("YuanDuanKou")%>' min-width="120" prop="SourcePort" show-overflow-tooltip></el-table-column>
                                <el-table-column label='Target Port' min-width="120" prop="TargetPort" show-overflow-tooltip></el-table-column>
                                <el-table-column label='Action' min-width="120" prop="OperationType" show-overflow-tooltip></el-table-column>
                            </el-ctable>
                        </div>
                    </div>
                </div>
                <div class='itemMainBoxFooter'>
                    <el-button type="primary" @click="submitLinuxVppConfig" style="margin-left:20px;"><%=rb.getString("QueDing")%></el-button>
                    <el-button @click="closeLinkSetting"><%=rb.getString("QuXiao")%></el-button>
                </div>
            </el-tab-pane>
            <el-tab-pane label="VPP" name="vpp">
                <div class="egwTabPaneContent">
                    <div class="egwTabPaneItemBoxCls">
                        <div class="itemMainBoxTitle">
                            <span class="el-icon el-icon-splitGroup"></span>
                            Set Route
                            <span class="el-icon el-icon-circle-add" style="position:absolute;right:55px;" @click="routeAdd('vpp')"></span>
                        </div>
                        <div class="egwTabPaneTableBoxCls">
                            <el-ctable 
                                ref="vppRouteListTable"
                                :rownumber="true" 
                                id="vppRouteListTable" 
                                :url="vppRouteListTableUrl"
                                :time="6"
                                height="100%" 
                                pagination="true"
                             >
                                <el-table-column label='<%=rb.getString("CaoZuo")%>' width="100" fixed>
                                    <template slot-scope="scope">
                                        <span class="el-icon el-icon-circle-warning yellowIcon" title="Added not Send" v-if="scope.row.dataStatus == '2'"></span>
                                        <span class="el-icon el-icon-circle-warning redIcon" title="Send Failure" v-if="scope.row.dataStatus == '4'"></span>
                                        <span class="el-icon el-icon-operation-delete" @click="delRoute(scope.row,'vpp',event)" style="margin-left:5px;"></span>
                                    </template>
                                </el-table-column>
                                <el-table-column label='TargetIP' min-width="120" prop="TargetIp" show-overflow-tooltip></el-table-column>
                                <el-table-column label='NextHopIP' min-width="120" prop="NextHop" show-overflow-tooltip></el-table-column>
                                <el-table-column label='Device' min-width="120" prop="Device" show-overflow-tooltip></el-table-column>
                            </el-ctable>
                        </div>
                    </div>
                    <div class="egwTabPaneItemBoxCls">
                        <div class="itemMainBoxTitle">
                            <span class="el-icon el-icon-splitGroup"></span>
                            ACL Configuration
                            <span class="el-icon el-icon-circle-add" style="position:absolute;right:55px;" @click="aclAdd('vpp')"></span>
                        </div>
                        <div class="egwTabPaneTableBoxCls">
                            <el-ctable 
                                ref="vppAclListTable" 
                                :rownumber="false" 
                                id="vppAclListTable" 
                                :url="vppAclListTableUrl"
                                :time="6"
                                height="100%"
                                pagination="true"
                             >
                                <el-table-column label='<%=rb.getString("CaoZuo")%>' width="100" fixed>
                                    <template slot-scope="scope">
                                        <span class="el-icon el-icon-circle-warning yellowIcon" title="Added not Send" v-if="scope.row.dataStatus == '2'"></span>
                                        <span class="el-icon el-icon-circle-warning redIcon" title="Send Failure" v-if="scope.row.dataStatus == '4'"></span>
                                        <span class="el-icon el-icon-operation-delete" @click="delAcl(scope.row,'vpp',event)" style="margin-left:5px;"></span>
                                    </template>
                                </el-table-column>
                                <el-table-column label='<%=rb.getString("WangKaMingCheng")%>' min-width="150" prop="NetCardName"></el-table-column>
                                <el-table-column label='<%=rb.getString("ShuJuFangXiang")%>' min-width="120" prop="direction" show-overflow-tooltip></el-table-column>
                                <el-table-column label='<%=rb.getString("YuanIP")%>' min-width="100" prop="SourceIp"></el-table-column>
                                <el-table-column label='Target IP' min-width="120" prop="TargetIp" show-overflow-tooltip></el-table-column>
                                <el-table-column label='<%=rb.getString("XieYiLeiXing")%>' min-width="120" prop="ProtocolCode"></el-table-column>
                                <el-table-column label='<%=rb.getString("YuanDuanKou")%>' min-width="120" prop="SourcePort" show-overflow-tooltip></el-table-column>
                                <el-table-column label='Target Port' min-width="120" prop="TargetPort" show-overflow-tooltip></el-table-column>
                                <el-table-column label='Action' min-width="120" prop="OperationType" show-overflow-tooltip></el-table-column>
                            </el-ctable>
                        </div>
                    </div>
                </div>
                <div class='itemMainBoxFooter'>
                    <el-button type="primary" @click="submitLinuxVppConfig" style="margin-left:20px;"><%=rb.getString("QueDing")%></el-button>
                    <el-button @click="closeLinkSetting"><%=rb.getString("QuXiao")%></el-button>
                </div>
            </el-tab-pane>
		</el-tabs>
	</div>
    <!-- 路由新增 弹窗 -->
	<el-dialog class="gnbConfigAddDialog" top="25vh" title="<%=rb.getString("TianJia")%>" width="50%" :visible.sync="addRouteDialogShow"  @close="closeAddRoute" :close-on-click-modal="false" :modal-append-to-body="false">		
		<el-form ref="addRouteForm" :model='addRouteForm' :rules='addRouteRules' label-position="top">     		     			            
			<el-form-item label="TargetIP" style="min-width:400px;" prop="TargetIp" class='validate-item'>
				<el-input v-model.trim='addRouteForm.TargetIp' placeholder='<%=rb.getString("IPMaskGeShiTiShi")%>'>
                    <template slot="append">Support IPv4 or IPv6</template>
                </el-input>
			</el-form-item>
			<el-form-item label="NextHopIP" style="min-width:400px;" prop="NextHop" class='validate-item'>
				<el-input v-model.trim='addRouteForm.NextHop' placeholder='Example:192.168.9.0'>
                    <template slot="append">Support IPv4 or IPv6</template>
                </el-input>
			</el-form-item>
         	<el-form-item label="Device" style="min-width:400px;" prop="Device">
         		<el-input v-model.trim='addRouteForm.Device'></el-input>
         	</el-form-item>
        </el-form> 
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="addRouteSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="addRouteDialogShow = false"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
    <!--  ACL 新增 弹窗 -->
	<el-dialog class="gnbConfigAddDialog" top="25vh" title='<%=rb.getString("TianJia")%>' width="50%" :visible.sync="addAclDialogShow" @close="closeAddAcl" :close-on-click-modal="false" :modal-append-to-body="false">		
		<el-form ref="addAclForm" :model='addAclForm' :rules='addAclRules' label-position="top">
            <el-form-item label='<%=rb.getString("WangKaMingCheng")%>' :required="addAclType == 'vpp'" style="min-width:400px;"  prop="NetCardName">
				<el-input v-model.trim='addAclForm.NetCardName'></el-input>
			</el-form-item>
            <el-form-item label='<%=rb.getString("ShuJuFangXiang")%>' style="min-width:400px;"  prop="direction">
                <el-radio-group v-model="addAclForm.direction" size="small">
                    <el-radio border label="input">Input</el-radio>
                    <el-radio border label="output">Output</el-radio>
                </el-radio-group>
			</el-form-item>
            <el-form-item label='<%=rb.getString("YuanIP")%>' style="min-width:400px;"  prop="SourceIp" class='validate-item'>
				<el-input v-model.trim='addAclForm.SourceIp' placeholder='<%=rb.getString("IPMaskGeShiTiShi")%>'>
                    <template slot="append">Support IPv4 or IPv6</template>
                </el-input>
			</el-form-item>
            <el-form-item label='<%=rb.getString("XieYiLeiXing")%>' style="min-width:400px;"  prop="ProtocolCode">
                <el-select v-model='addAclForm.ProtocolCode'>
                    <el-option label="ICMP" value="icmp"></el-option>
                    <el-option label="UDP" value="udp"></el-option>
                    <el-option label="TCP" value="tcp"></el-option>
                    <el-option label="SCTP" value="sctp"></el-option>
                </el-select>
         	</el-form-item>
            <el-form-item v-if="!(addAclType == 'linux' && addAclForm.ProtocolCode == 'icmp')" label='<%=rb.getString("YuanDuanKou")%>' style="min-width:400px;"  prop="SourcePort" class='validate-item'>
                <span slot="label" class="labelIconCls">
                    <%=rb.getString("YuanDuanKou")%>(0~65535)
                </span>
				<el-input v-model.trim='addAclForm.SourcePort'>
                    <template slot="append">Example：8000 or 8000-8600</template>
                </el-input>
			</el-form-item>  	     			            
			<el-form-item label="Target IP" style="min-width:400px;"  prop="TargetIp" class='validate-item'>
				<el-input v-model.trim='addAclForm.TargetIp' placeholder='<%=rb.getString("IPMaskGeShiTiShi")%>'>
                    <template slot="append">Support IPv4 or IPv6</template>
                </el-input>
			</el-form-item>
            <el-form-item v-if="!(addAclType == 'linux' && addAclForm.ProtocolCode == 'icmp')" label="Target Port" style="min-width:400px;"  prop="TargetPort" class='validate-item'>
                <span slot="label" class="labelIconCls">
                    Target Port(0~65535)
                </span>
				<el-input v-model.trim='addAclForm.TargetPort'>
                    <template slot="append">Example：8000 or 8000-8600</template>
                </el-input>
			</el-form-item>
            <el-form-item label='Action' style="min-width:400px;"  prop="OperationType">
                <el-radio-group v-model="addAclForm.OperationType" size="small">
                    <el-radio border label="permit">Permit</el-radio>
                    <el-radio border label="deny">Deny</el-radio>
                </el-radio-group>
			</el-form-item>
        </el-form> 
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="addAclSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="addAclDialogShow = false"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
</div>

<script>
var regIp = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/,
    regKey = /^[A-Fa-f0-9]{32}$/,
    regNumber = /^[0-9]{15}$/;
var egwSystemConfigPage = new Vue({
	el: '#egwSystemConfigPage', 
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
                    if(this.addAclType == 'linux' && this.addAclForm.ProtocolCode == 'icmp'){
                        callback();
                    }else{
                        callback(new Error('Range example：8000 or 8000-8600'))
                    }
                }else{
                    if(vm.isCorrectPort(value)){
                        callback();
                    }else{
                       callback(new Error('Range example：8000 or 8000-8600'))
                    }
                }
			},
            validateNetCardName= function(rule,value,callback) {
                
                if( value === '' || value === null || value === undefined) {
                    if(this.addAclType == 'vpp'){
                        callback(new Error('<%=rb.getString("BiTian")%>'))
                    }else{
                        callback();
                    }
                }else {
                    callback();
                }
            };
		return {
            egwCode:'',
            egwSn:'',
            activeName:'Basic',
            addRouteType:'',
            addAclType:'',
            commonConfigForm:{
                egwName:'',
                egwDescription:'',
            },
            addRouteDialogShow:false,
            addRouteForm:{
                TargetIp:'',
                NextHop:'',
                Device:'',
            },
            addRouteRules:{
                TargetIp:[{required:true,message:'<%=rb.getString("IPMaskGeShiTiShi")%>',validator: validateIPorMask}],
                NextHop:[{required:true,validator: validateRouteIP}]
            },
            addAclDialogShow:false,
            addAclForm:{
                NetCardName:'',
                SourceIp:'',
                SourcePort:'',
                TargetIp:'',
                TargetPort:'',
                OperationType:'permit',
                direction:'input',
                ProtocolCode:'icmp',
            },
            addAclRules:{
                NetCardName:{validator: validateNetCardName},
                SourceIp:[{required:true,message:'<%=rb.getString("IPMaskGeShiTiShi")%>',validator: validateIPorMask}],
                SourcePort:[{message:'Range example：8000 or 8000-8600',validator: validatePort}],
                TargetIp:[{required:true,message:'<%=rb.getString("IPMaskGeShiTiShi")%>',validator: validateIPorMask}],
                TargetPort:[{message:'Range example：8000 or 8000-8600',validator: validatePort}],
            },
            linuxRouteListTableUrl:'',
            linuxAclListTableUrl:'',
			vppRouteListTableUrl:'',
            vppAclListTableUrl:'',
		};
	},
	computed: {},
	methods: {
        // 初始化
		init(row,code,sn,status){
		    var vm =this;
			vm.egwCode = code;
            vm.egwSn = sn;
            vm.getCommonConfigInfo();
		},
        // Tab 切换
		tabClick(){
            var vm = this,
                str = Math.random().toString();
            if(vm.activeName == 'linux'){
                vm.linuxRouteListTableUrl = '${ctx}/egw/config/getRouteList.action?type=linux&egwCode='+ vm.egwCode + '&randomCode=' + str;
                vm.linuxAclListTableUrl = '${ctx}/egw/config/getACLList.action?type=linux&egwCode='+ vm.egwCode + '&randomCode=' + str;
            }else{
                vm.vppRouteListTableUrl = '${ctx}/egw/config/getRouteList.action?type=vpp&egwCode='+ vm.egwCode + '&randomCode=' + str;
                vm.vppAclListTableUrl = '${ctx}/egw/config/getACLList.action?type=vpp&egwCode='+ vm.egwCode + '&randomCode=' + str;
            }
        },
        // 基础配置 回显
        getCommonConfigInfo(){
            var vm = this,
                code = vm.egwCode,
                params={
                    egwCode:code
                };
            axios.post('${ctx}/egw/config/getCommonConfig.action',stringify(params)).then(function(response){
                var data = response.data;
                if(data){
                    Object.keys(vm.commonConfigForm).forEach(function(key){
                        if(data[key]){
                            vm.commonConfigForm[key] = data[key];
                        }
                    });
                    initForm(vm.$refs.commonConfigForm);
                }
            }).catch(function(error){});

        },
        // 基础配置 提交
        submitCommonConfig(){
            var vm = this,
                urls = '${ctx}/egw/config/setCommonConfig.action',
                params = {
                    egwCode:vm.egwCode
                },
                isChanged = isFormChanged(vm.$refs.commonConfigForm);
            if(!isChanged){
                showMsg('prompt_msg','<%=rb.getString("CanShuZhiMeiYouBianHua")%>');
                return;
            }
            vm.$refs.commonConfigForm.fields.map(function(field){
                if(Array.isArray(field.fieldValue)){
                    
                    var vList = field.fieldValue.map(function(item){return item}),
                        oList = (field.reinitialValue||[]).map(function(item){return item}),
                        val = JSON.stringify(vList.sort()),
                        orVal = JSON.stringify(oList.sort());

                    if(val != orVal) {
                        var editList=[],subList=[];
                        if(val != orVal) {
                            editList.push(field.prop);
                        }
                    };
                }else{
                    if(vm.isNull(field.fieldValue) && vm.isNull(field.reinitialValue)){
                            
                    }else if(field.fieldValue != field.reinitialValue) {
                        params[field.prop] = field.fieldValue;
                    };
                }
            });
            if(params.egwName){
                params.egwName = encodeURIComponent(params.egwName);
            }
            if(params.egwDescription){
                params.egwDescription = encodeURIComponent(params.egwDescription);
            }
            vm.$refs["commonConfigForm"].validate( valid => {
                if(valid){
                    axios.post(urls,stringify(params)).then(res=>{
                        var data = res.data;
                        if(data["success"]){
                            vm.$message({
                                message:'<%=rb.getString("YiXiaFaTiShi")%>',
                                type:'success',
                            })
                            vm.closeLinkSetting();
                        }else{
                            vm.$message.error(data["message"])
                        }
                    })
                }else{
                    return false
                }
            }) 
        },
        // 关闭配置页面
        closeLinkSetting(){
            var vm = this;
            egwMonitor.$refs.egwSettingPage.hide();
        },
        // Route 新增
        routeAdd(type){
            var vm = this;
            vm.addRouteType = type;
			vm.addRouteDialogShow = true;
        },
        // 路由新增 新增提交
        addRouteSubmit(){
            var vm = this,
                urls="${ctx}/egw/config/addRouteInfo.action",
                params={
                    egwCode:vm.egwCode,
                    type:vm.addRouteType
                };
            Object.keys(vm.addRouteForm).forEach(function(key){
                if(vm.addRouteForm[key]){
                    params[key] = vm.addRouteForm[key];
                }
            });
            vm.$refs["addRouteForm"].validate( valid => {
                if(valid){
                    axios.post(urls,stringify(params)).then(res=>{
                        var data = res.data;
                        if(data["success"]){
                            vm.addRouteDialogShow = false;
                            if(vm.addRouteType == 'linux'){
                                vm.$refs.linuxRouteListTable.refresh();
                            }else{
                                vm.$refs.vppRouteListTable.refresh();
                            }
                        }else{
                            vm.$message.error(data["message"])
                        }
                    })
                }else{
                    return false
                }
            })
        },
        // 路由新增 新增取消
        closeAddRoute(){
            var vm = this,
                params={
                    TargetIp:'',
                    NextHop:'',
                    Device:'',
                };
            Object.assign(vm.addRouteForm,params);
            vm.$refs["addRouteForm"].clearValidate();
        },
        // 路由新增 删除事件
        delRoute(row,type){
            var vm = this,
                urls="${ctx}/egw/config/deleteRouteInfo.action",
                params={
                    egwCode:vm.egwCode,
                    type:type
                };
            vm.addRouteType = type;
            params.dataId = row.dataId;
            vm.$confirm('<%=rb.getString("QueDingShanChuRenWu")%>','<%=rb.getString("QueRen")%>',{
                customClass:'warningConfirm',
                confirmButtonText:'<%=rb.getString("QueDing")%>',
                cancalButtonText:'<%=rb.getString("QuXiao")%>',
                dangerouslyUseHTMLString:true
            }).then(()=>{
                axios.post(urls,stringify(params)).then(res=>{
                    var data = res.data;
                    if(data["success"]){
                        if(vm.addRouteType == 'linux'){
                            vm.$refs.linuxRouteListTable.refresh();
                        }else{
                            vm.$refs.vppRouteListTable.refresh();
                        }
                    }else{
                        vm.$message.error(data["message"])
                    }
                })
            }).catch(()=>{})
        },
        // ACL 新增
        aclAdd(type){
             var vm = this;
            vm.addAclType = type;
			vm.addAclDialogShow = true;
        },
        // ACL 新增提交
        addAclSubmit(){
            var vm = this,
                urls="${ctx}/egw/config/addAclInfo.action",
                params={
                    egwCode:vm.egwCode,
                    type:vm.addAclType
                };
            Object.keys(vm.addAclForm).forEach(function(key){
                if(vm.addAclForm[key]){
                    params[key] = vm.addAclForm[key];
                }
            });
            if(vm.addAclType == 'linux' && vm.addAclForm.ProtocolCode == 'icmp'){
                delete params.SourcePort
                delete params.TargetPort
            }
            vm.$refs["addAclForm"].validate( valid => {
                if(valid){
                    axios.post(urls,stringify(params)).then(res=>{
                        var data = res.data;
                        if(data["success"]){
                            vm.addAclDialogShow = false;
                            if(vm.addAclType == 'linux'){
                                vm.$refs.linuxAclListTable.refresh();
                            }else{
                                vm.$refs.vppAclListTable.refresh();
                            }
                        }else{
                            vm.$message.error(data["message"])
                        }
                    })
                }else{
                    return false
                }
            })
        },
        // ACL  新增取消
        closeAddAcl(){
            var vm = this,
                params={
                    NetCardName:'',
                    SourceIp:'',
                    SourcePort:'',
                    TargetIp:'',
                    TargetPort:'',
                    OperationType:'permit',
                    direction:'input',
                    ProtocolCode:'icmp',
                };
            Object.assign(vm.addAclForm,params);
            vm.$refs["addAclForm"].clearValidate();
        },
        // ACL  删除事件
        delAcl(row,type){
            var vm = this,
                urls="${ctx}/egw/config/deleteAclInfo.action",
                params={
                    egwCode:vm.egwCode,
                    type:type
                };
            vm.addAclType = type;
            params.dataId = row.dataId;
            vm.$confirm('<%=rb.getString("QueDingShanChuRenWu")%>','<%=rb.getString("QueRen")%>',{
                customClass:'warningConfirm',
                confirmButtonText:'<%=rb.getString("QueDing")%>',
                cancalButtonText:'<%=rb.getString("QuXiao")%>',
                dangerouslyUseHTMLString:true
            }).then(()=>{
                axios.post(urls,stringify(params)).then(res=>{
                    var data = res.data;
                    if(data["success"]){
                        if(vm.addAclType == 'linux'){
                            vm.$refs.linuxAclListTable.refresh();
                        }else{
                            vm.$refs.vppAclListTable.refresh();
                        }
                    }else{
                        vm.$message.error(data["message"])
                    }
                })
            }).catch(()=>{})
        },
        submitLinuxVppConfig(){
            var vm = this,
                urls="${ctx}/egw/config/saveLinuxVppInfos.action",
                params={
                    egwCode:vm.egwCode,
                    type:vm.activeName
                };
            axios.post(urls,stringify(params)).then(res=>{
                var data = res.data;
                if(data["success"]){
                    vm.$message({
                        message: '<%=rb.getString("YiXiaFaTiShi")%>',
                        type:'success',
                    });
                }else{
                    vm.$message.error(data["message"])
                }
            })
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
        // Acl 端口校验  支持 单个及范围值
        isCorrectPort(value){
            var vm = this,
                PortList = value.split('-');
            if(PortList.length > 2){
                return false
            }else if(PortList.length < 2){
                if(vm.isInteger(value)&& parseInt(value)>=0 && parseInt(value)<=65535){
                    return true
                }else{
                    return false
                }
            }else{
                if(vm.isCorrectRange(value)){
                     return true
                }else{
                    return false
                }
            }
        },
        // 是否符合例如：1-10 正确范围值
		isCorrectRange(val){
			var min = parseFloat(val.split('-')[0]),
				max = parseFloat(val.split('-')[1]);
			if(min >= max ){
				return false
			}
            if(min < 0){
				return false
			}
			if(max > 65535){
				return false
			}
			return true
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
         // 判断是否为空
        isNull(val){
            if(val==undefined || val == null || val =="") return true;
            else return false;
        },
        // 同步
        syncSubmit(){
            var vm = this,
                urls='${ctx}/egw/config/refreshConfig.action',
                params={
                    egwCode: vm.egwCode,
                    refreshType: 'SYSTEMCONFIG'
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
		eventBus.$off("egw-data").$on("egw-data",this.init)
	}
});

</script>