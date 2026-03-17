<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
#egw5GSignalingGatewayPage{
    width:100%;
    height:calc(100% - 10px);
    position: relative;
}
#egw5GSignalingGatewayPage .itemMainBoxCls{
	border:1px solid #d5dcec;
	border-radius:10px;
	margin:8px;
	background:#fff;
	height:100%;
}
#egw5GSignalingGatewayPage .itemMainBoxCls .el-tab-pane{
	position: relative;
}
#egw5GSignalingGatewayPage .itemMainBoxTitle {
	height:36px;
    line-height: 36px;
	font-size:14px;
	font-weight:bold;
    position: relative;
}
#egw5GSignalingGatewayPage .egwTabPaneItemBoxCls{
	flex: 1;
}
#egw5GSignalingGatewayPage .egwTabPaneTableBoxCls{
    height:calc(100% - 120px);
    overflow: hidden;
    box-sizing: border-box;
    border:1px solid #d5dcec;
    border-radius: 4px;
    margin: 0px 50px 0px 20px;
}
#egw5GSignalingGatewayPage .toolbarBoxCls{
    padding: 10px 0px;
}
#egw5GSignalingGatewayPage .egwTabPaneContent{
    margin: 25px 0px 0px 40px;
    display: flex;
    flex-direction: column;
    height: calc(100% - 90px);
    overflow: auto;
}
#egw5GSignalingGatewayPage .egwTabPaneContent .egwTabPaneItemCls{
    margin-bottom: 20px;
    font-size: 14px;
}
#egw5GSignalingGatewayPage .itemMainBoxFooter{
    display: flex;
    align-items: center;
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
#egw5GSignalingGatewayPage .plmnBoxCls,
#egw5GSignalingGatewayPage .ipAndPortBoxCls,
#egw5GSignalingGatewayPage .upLinkIpAndDownLinkIpBoxCls,
#egw5GSignalingGatewayPage .cellItemBoxCls{
    display: flex;
    flex-wrap: wrap;
}
#egw5GSignalingGatewayPage .leftAndRightItemCls{
    width: 50%;
    min-width: 500px;
}
#egw5GSignalingGatewayPage .disabledIconBox .el-icon::before{
    color: #e9e9e9;
}
#egw5GSignalingGatewayPage .mostNumberCls{
    color: #999999;
    margin-left: 10px;
}
#egw5GSignalingGatewayPage .itemListBoxCls{
    padding-left: 25px;
    padding-top: 5px;
}
#egw5GSignalingGatewayPage .itemCls{
    height: 24px;
    display: inline-block;
    line-height: 24px;
    border: 1px solid #4D84FF;
    box-sizing: border-box;
    padding: 0px 10px;
    margin-right: 10px;
    margin-bottom: 10px;
}
#egw5GSignalingGatewayPage .itemListBoxCls .el-icon-close{
    font-size: unset;
    position: unset;
    top: unset;
    right: unset;
}
#egw5GSignalingGatewayPage .itemListBoxCls .el-icon-close::before{
    font-size: 14px;
}
#egw5GSignalingGatewayPage .errorBoxCls{
    color:red;
    font-size:10px;
}
#egw5GSignalingGatewayPage .egwTabPaneContent .el-input__suffix{
    height: 26px;
    display: flex;
    align-items: center;
}
#egw5GSignalingGatewayPage .cellItemBoxCls .el-select>.el-input{
   width: 230px;
}
#egw5GSignalingGatewayPage .bottomLine{
    background-color:#E9E9E9;
    width: 100%;
    height: 1px;
    margin-bottom: 10px;
}
#egw5GSignalingGatewayPage .gnbSectionListBoxCls{
    height: 260px;
    margin-right: 40px;
}
.gnbConfigAddDialog .el-input-group__append{
    border-radius:0px;
    border-right:none;
}
.gnbConfigAddDialog .validate-item .el-input-group__append{
    border:none;
    background:none;
}
.gnbConfigAddDialog .validate-item .el-form-item__error{
    display:none;
}
.gnbConfigAddDialog .is-error .el-input-group__append{
    color:#FA5555;
}
.gnbConfigAddDialog .el-form-item__error{
    padding-top: 0px;
}
.gnbConfigAddDialog .validate-item .el-input__inner{
    width:200px;
}
.gnbConfigAddDialog .el-dialog__header .el-icon-close{
    top: 0px;
    font-size: 18px;
}
.gnbConfigAddDialog  .selectAndInputBoxCls{
    position:relative
}
.gnbConfigAddDialog  .selectAndInputBoxCls .el-input__suffix{
    z-index: 666
}
.gnbConfigAddDialog .selectAndInput_input {
    padding-right: 30px;
    position:absolute;
    left:1px;
    top: 1px;
}
.gnbConfigAddDialog .selectAndInput_input .el-input__inner{
    width: 170px;
    height: 22px;
    border: none;
    margin-top: 1px;
}
.gnbConfigAddDialog .selectAndInput_input  .el-input-group__append{
    padding-left: 50px;
}
.sliceAndInterfacePopoverClass{
	padding: 10px!important;
}
</style>

<div id="egw5GSignalingGatewayPage">
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
                    <div style="margin-bottom:20px;">
                        <el-form ref="gnbSettingBasicForm" :model="gnbSettingBasicForm" :rules="gnbBasicFormRules" label-position="top"  :hide-required-asterisk='true'>
                            <div class="allowMoreInputBoxCls">
                                <div class="allowMoreInputHeadCls">
                                    <span class="allowMoreInputTitleCls">PLMN</span>
                                    <span class="allowMoreInputTipsCls">（Add 8 at most）</span>
                                </div>
                                <div class="allowMoreInputContentCls">
                                    <div class="allowMoreInputFieldCls">
                                        <el-input v-model="gnbSettingBasicForm.plmn" placeholder='<%=rb.getString("PLMNFanWei")%>'></el-input>
                                        <div class="allowMoreInputAddBtnCls" @click="addGnbPLMNs" v-show="gnbPlmnList.length < 8 ">
                                            <span class="el-icon el-icon-plus"></span>
                                            <span>Add</span>
                                        </div>
                                    </div>
                                    <div class="allowMoreInputParamsCls">
                                        <div v-for="item in gnbPlmnList" class="allowMoreInputParamsItemCls">
                                            <span>{{item.Hplmn}}</span>
                                            <span class="el-icon el-icon-close" style="margin-left:5px;" @click="gnbPlmnListDel(item)"></span>
                                        </div>
                                    </div>
                                </div>
                                <div class="allowMoreInputFootCls">
                                    <p class="inputErrorBoxCls">{{gnbPlmnErrorMessage}}</p>
                                </div>
                            </div>
                            <div class="allowMoreInputBoxCls">
                                <div class="allowMoreInputHeadCls">
                                    <span class="allowMoreInputTitleCls">NGAP IP to HgNB</span>
                                    <span class="allowMoreInputTipsCls">（Add 4 at most,Support IPv4 or IPv6）</span>
                                </div>
                                <div class="allowMoreInputContentCls">
                                    <div class="allowMoreInputFieldCls">
                                        <el-input v-model="gnbSettingBasicForm.ngAmfIp"></el-input>
                                        <div class="allowMoreInputAddBtnCls" @click="addNgAmfIp" v-show="gnbNgAmfIpList.length < 4 ">
                                            <span class="el-icon el-icon-plus"></span>
                                            <span>Add</span>
                                        </div>
                                    </div>
                                    <div class="allowMoreInputParamsCls">
                                        <div v-for="item in gnbNgAmfIpList" class="allowMoreInputParamsItemCls">
                                            <span>{{item.Ip}}</span>
                                            <span class="el-icon el-icon-close" style="margin-left:5px;" @click="gnbNgAmfIpListDel(item)"></span>
                                        </div>
                                    </div>
                                </div>
                                <div class="allowMoreInputFootCls">
                                    <p class="inputErrorBoxCls">{{ngAmfIpErrorMessage}}</p>
                                </div>
                            </div>
                            <el-form-item label="NGAP Port to HgNB" style="margin:10px 0px 10px 25px;">
                                <el-input style='width:230px;' v-model="gnbSettingBasicForm.ngAmfPort" :disabled="true"></el-input>
                            </el-form-item>
                            <div class="allowMoreInputBoxCls">
                                <div class="allowMoreInputHeadCls">
                                    <span class="allowMoreInputTitleCls">UpLink N3 IP to UPF</span>
                                    <span class="allowMoreInputTipsCls">（Add 1 at most,Support IPv4 or IPv6）</span>
                                </div>
                                <div class="allowMoreInputContentCls">
                                    <div class="allowMoreInputFieldCls">
                                        <el-input v-model="gnbSettingBasicForm.upLinkIp"></el-input>
                                        <div class="allowMoreInputAddBtnCls" @click="addGnbUpLinkIp" v-show="gnbUpLinkIpList.length < 1 ">
                                            <span class="el-icon el-icon-plus"></span>
                                            <span>Add</span>
                                        </div>
                                    </div>
                                    <div class="allowMoreInputParamsCls">
                                        <div v-for="item in gnbUpLinkIpList" class="allowMoreInputParamsItemCls">
                                            <span>{{item.N3bAddr}}</span>
                                            <span class="el-icon el-icon-close" style="margin-left:5px;" @click="gnbUpLinkIpListDel(item)"></span>
                                        </div>
                                    </div>
                                </div>
                                <div class="allowMoreInputFootCls">
                                    <p class="inputErrorBoxCls">{{gnbUpLinkIpErrorMessage}}</p>
                                </div>
                            </div>
                            <div class="allowMoreInputBoxCls">
                                <div class="allowMoreInputHeadCls">
                                    <span class="allowMoreInputTitleCls">Downlink N3 IP to HgNB</span>
                                    <span class="allowMoreInputTipsCls">（Add 1 at most,Support IPv4 or IPv6）</span>
                                </div>
                                <div class="allowMoreInputContentCls">
                                    <div class="allowMoreInputFieldCls">
                                        <el-input v-model="gnbSettingBasicForm.downLinkIp"></el-input>
                                        <div class="allowMoreInputAddBtnCls" @click="addGnbDownLinkIp" v-show="gnbDownLinkIpList.length < 1 ">
                                            <span class="el-icon el-icon-plus"></span>
                                            <span>Add</span>
                                        </div>
                                    </div>
                                    <div class="allowMoreInputParamsCls">
                                        <div v-for="item in gnbDownLinkIpList" class="allowMoreInputParamsItemCls">
                                            <span>{{item.N3aAddr}}</span>
                                            <span class="el-icon el-icon-close" style="margin-left:5px;" @click="gnbDownLinkIpListDel(item)"></span>
                                        </div>
                                    </div>
                                </div>
                                <div class="allowMoreInputFootCls">
                                    <p class="inputErrorBoxCls">{{gnbDownLinkIpErrorMessage}}</p>
                                </div>
                            </div>
                            <div class="cellItemBoxCls">
                                <div class="leftAndRightItemCls">
                                    <el-form-item prop="" label="HgNB ID Length"  label-width="160px" style="margin: 10px 0px 10px 25px;">
                                        <el-select v-model='gnbSettingBasicForm.GNB_idLength' style='width:230px;'>
                                            <el-option v-for="item in GNB_idLengthList" :label='item.label' :value='item.value' :disabled="item.disabled"></el-option>
                                        </el-select>
                                    </el-form-item>
                                </div>
                                <div class="leftAndRightItemCls">
                                    <el-form-item prop="" label="gNB ID Length"  label-width="160px" style="margin: 10px 0px 10px 25px;">
                                        <el-select v-model='gnbSettingBasicForm.AMF_idLength' style='width:230px;'>
                                            <el-option v-for="item in AMF_idLengthList" :label='item.label' :value='item.value' :disabled="item.disabled"></el-option>
                                        </el-select>
                                    </el-form-item>
                                </div>
                            </div>
                        </el-form>
                     </div>
                    <div class="bottomLine"></div>
                    <div class="itemMainBoxTitle">
                        <span class="el-icon el-icon-splitGroup"></span>
                        Slice List
                        <span class="el-icon el-icon-circle-add" @click="sectionAdd" style="position:absolute;right:55px;"></span>
                    </div>
                     <div class="gnbSectionListBoxCls">
                        <el-ctable 
                            ref="gnbSectionListTable"
                            :rownumber="true" 
                            id="gnbSectionListTable" 
                            :url="gnbSectionListTableUrl"
                            :time="6"
                            height="100%"
                            :pagination="true"
                            style="border:1px solid #E9E9E9;margin-left:25px;"
                        >
                            <el-table-column label='<%=rb.getString("CaoZuo")%>' width="100">
                                <template slot-scope="scope">
                                    <span class="el-icon el-icon-operation-delete" @click="delSection(scope.row,event)" style="margin-left:20px;"></span>
                                </template>
                            </el-table-column>
                            
                            <el-table-column label='HPLMN' min-width="120" prop="plmn" show-overflow-tooltip></el-table-column>
                            <el-table-column label='TAC' min-width="120" prop="tac" show-overflow-tooltip></el-table-column>
                            <el-table-column label='SST' min-width="120" prop="sst" show-overflow-tooltip></el-table-column>
                            <el-table-column label='SD' min-width="120" prop="sd" show-overflow-tooltip></el-table-column>
                        </el-ctable>
                    </div>
                </div>
                <div class='itemMainBoxFooter'>
                    <el-button type="primary" @click="submitGnbBasicAndEnbList" style="margin-left:20px;"><%=rb.getString("QueDing")%></el-button>
                    <el-button @click="closeSettingPage"><%=rb.getString("QuXiao")%></el-button>
                </div>
            </el-tab-pane>
            <el-tab-pane label="Link Configuration" name="gnbConfig">
                <div class="egwTabPaneContent">
                    <div class="itemMainBoxTitle">
                        <span class="el-icon el-icon-splitGroup"></span>
                        gNodeB ID List
                        <span class="el-icon el-icon-circle-add" @click="linkAdd('gnb')" style="position:absolute;right:55px;"></span>
                    </div>
                    <div class="egwTabPaneTableBoxCls">
                        <el-ctable 
                            ref="gnbIdMmeListTable" 
                            :rownumber="true" 
                            id="gnbIdMmeListTable" 
                            :data="gnbIdMmeListTableData" 
                            height="100%"
                            :front-pagination="true"
                            :pagination="true"
                         >
                            <el-table-column label='<%=rb.getString("CaoZuo")%>' min-width="100">
                                <template slot-scope="scope">
                                    <span class="el-icon el-icon-operation-edit" @click="editGnbIdMme(scope.row,event)"></span>
                                    <span class="el-icon el-icon-operation-delete" @click="delGnbIdMme(scope.row,event)" style="margin-left:20px;"></span>
                                </template>
                            </el-table-column>
                            <el-table-column label='gNodeB ID' min-width="120" prop="enodebId" show-overflow-tooltip></el-table-column>
                            <el-table-column label='Link Num' min-width="120" prop="linkNum" show-overflow-tooltip></el-table-column>
                            <el-table-column label='PLMN' min-width="120" prop="Hplmn" show-overflow-tooltip></el-table-column>
                            <el-table-column label='TAC' min-width="120" prop="Tac" show-overflow-tooltip></el-table-column>
                        </el-ctable>
                    </div>
                </div>
                <div class='itemMainBoxFooter'>
                    <el-button type="primary" @click="submitGnbBasicAndEnbList" style="margin-left:20px;"><%=rb.getString("QueDing")%></el-button>
                    <el-button @click="closeSettingPage"><%=rb.getString("QuXiao")%></el-button>
                </div>
            </el-tab-pane>
            <el-tab-pane label="User Plane Configuration" name="PFM">
                <div class="egwTabPaneContent">
                    <div class="itemMainBoxTitle">
                        <span class="el-icon el-icon-splitGroup"></span>
                        <span style="font-size:14px;font-weight:bold;margin-right:10px;">Set Interface</span>
                        <span style="font-size:12px;color:#999999">No more than 8</span>
                        <span v-if="gnbInterfaceAddShow" class="el-icon el-icon-circle-add" @click="interfaceAdd" style="position:absolute;right:55px;"></span>
                    </div>
                    <div class="egwTabPaneTableBoxCls">
                       <el-ctable 
                            ref="gnbInterfaceListTable"
                            :rownumber="true" 
                            id="gnbInterfaceListTable" 
                            :url="gnbInterfaceListTableUrl"
                            @load-success="gnbInterfaceLoadSuccess"  
                            height="100%"
                            :pagination="true"
                        >
                            <el-table-column label='<%=rb.getString("CaoZuo")%>' width="100">
                                <template slot-scope="scope">
                                    <span class="el-icon el-icon-operation-delete" @click="delInterface(scope.row,event)" style="margin-left:20px;"></span>
                                </template>
                            </el-table-column>
                            
                            <el-table-column label='Name' min-width="120" prop="name" show-overflow-tooltip></el-table-column>
                            <el-table-column label='Type' min-width="120" prop="type" show-overflow-tooltip>
                                <template slot-scope="scope">
                                    <span v-if="scope.row.type == 'N3a'">To HgNB</span>
                                    <span v-if="scope.row.type == 'N3b'">To UPF</span>
                                </template>
                            </el-table-column>
                            <el-table-column label='IP' min-width="120" prop="ipAndMask" show-overflow-tooltip></el-table-column>
                            <el-table-column label='MTU' min-width="120" prop="mtu" show-overflow-tooltip></el-table-column>
                            <el-table-column label='Reassembly' min-width="120" prop="reassemblySwitch" show-overflow-tooltip></el-table-column>
                        </el-ctable>
                    </div>
                </div>
                <div class='itemMainBoxFooter' v-if="activeName == 'Basic' || activeName == 'gnbConfig'">
                    <el-button type="primary" @click="submitGnbBasicAndEnbList" style="margin-left:20px;"><%=rb.getString("QueDing")%></el-button>
                    <el-button @click="closeSettingPage"><%=rb.getString("QuXiao")%></el-button>
                </div>
            </el-tab-pane>
		</el-tabs>
	</div>
    <!-- 切片设置新增 弹窗 -->
	<el-dialog class="gnbConfigAddDialog" top="25vh" title="<%=rb.getString("TianJia")%>" width="1100px" :visible="addSectionDialogShow" :close-on-click-modal="false" :modal-append-to-body="false" @close="closeAddSection">		
		<div style="position:relative;top:-50px;left:30px;font-size:12px;color:#9E9E9E;width:800px;"><%=rb.getString("QiePianXinZengTiShi")%></div>
		<el-form label-position="top" ref="addSectionForm" :model='addSectionForm' :rules='addSectionRules' label-position="top">     		     			            
			<el-form-item label="HPLMN" style="min-width:400px;"  prop="plmn" style="margin-left:20px;" class='validate-item'>
				<el-input v-model.trim='addSectionForm.plmn'>
					<template slot="append">Length：5~6 Digit,Integer</template>
				</el-input>
			</el-form-item>
         	<el-form-item label="TAC" style="min-width:400px;"  prop="tac" class='validate-item'>
         		<el-input v-model.trim='addSectionForm.tac'>
					<template slot="append">Range:1-16777215,except 16777214</template>
				</el-input>
         	</el-form-item>   
			 <el-form-item label="SST" style="min-width:400px;"  prop="sst" class='validate-item selectAndInputBoxCls'>
				<el-select v-model='addSectionForm.sst'>
					<el-option label='1--eMBB' value='1--eMBB'></el-option>
					<el-option label='2--URLLC' value='2--URLLC'></el-option>
					<el-option label='3--MIoT' value='3--MIoT'></el-option>
					<el-option label='4--V2X' value='4--V2X'></el-option>
					<el-option label='5--HMTC' value='5--HMTC'></el-option>
				</el-select>
				<el-input class="selectAndInput_input" v-model.trim='addSectionForm.sst' maxlength="10">
					<template slot="append" style="margin-left:30px;">Custom Range：128~255</template>
				</el-input>
         	</el-form-item>   
			<el-form-item label="SD" style="min-width:400px;"  prop="sd" class='validate-item'>
         		<el-input v-model.trim='addSectionForm.sd'>
					<template slot="append">Range：0~16777215</template>
				</el-input>
         	</el-form-item>         	    
        </el-form> 
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="addSectionSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="closeAddSection"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
    <!-- PMF 连接设置新增 弹窗 -->
	<el-dialog class="gnbConfigAddDialog" top="25vh" title="<%=rb.getString("TianJia")%>" width="860px" :visible="addInterfaceDialogShow" :close-on-click-modal="false" :modal-append-to-body="false" @close="closeAddInterface">		
		<el-form label-position="top" ref="addInterfaceForm" :model='addInterfaceForm' :rules='addInterfaceRules' label-position="top">     		     			            
			<el-form-item label="Name" style="min-width:400px;"  prop="name" style="margin-left:20px;">
				<el-select v-model='addInterfaceForm.name'>
					<el-option v-for="(item,index) in CardList" :label='item.NetCardDiscrip' :value='item.NetCardDiscrip'></el-option>
				</el-select>
			</el-form-item>
			<el-form-item label="Type" style="min-width:400px;"  prop="type">
				<el-select v-model='addInterfaceForm.type'>
					<el-option label='To HgNB' value='N3a'></el-option>
					<el-option label='To UPF' value='N3b'></el-option>
				</el-select>
         	</el-form-item>   
         	<el-form-item label="IP" style="min-width:400px;"  prop="ipAndMask" class='validate-item'>
         		<el-input v-model.trim='addInterfaceForm.ipAndMask' placeholder='<%=rb.getString("IPMaskGeShiTiShi")%>'>
					 <template slot="append">Support IPv4 or IPv6</template>
				 </el-input>
         	</el-form-item>
			<el-form-item label="MTU" style="min-width:400px;"  prop="mtu" class='validate-item'>
         		<el-input v-model.trim='addInterfaceForm.mtu'>
					<template slot="append"><%=rb.getString("FanWei")%>：64~9220</template>
				</el-input>
         	</el-form-item>       
			 <el-form-item label="Reassembly" style="min-width:400px;"  prop="reassemblySwitch">
				<el-select v-model='addInterfaceForm.reassemblySwitch'>
					<el-option label='ON' value='on'></el-option>
					<el-option label='OFF' value='off'></el-option>
				</el-select>
         	</el-form-item>   
        </el-form> 
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="addInterfaceSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="closeAddInterface"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
    <el-slide ref="egwSettingLinkSlide" id="egwSettingLinkSlide" :url='settingLinkSlideUrl' :title="settingLinkSlideTitle" :footer="settingLinkSlideFooter" :header="settingLinkSlideHeader" 
        :position="settingLinkSlidePosition" :height="settingLinkSlideHeight"  :width='settingLinkSlideWidth' @ok="settingLinkSubmit"  @cancel="closeSettingLinkSlide" :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
    </el-slide>
</div>

<script>
var regIp = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/,
    regKey = /^[A-Fa-f0-9]{32}$/,
    regNumber = /^[0-9]{15}$/;
var egw5GSignalingGatewayPage = new Vue({
	el: '#egw5GSignalingGatewayPage', 
	data() {
		var vm = this;
        var validateSectionPLMN = (rule,value,callback) => {
                var reg = /^[0-9]{5,6}$/

                if(value === ''){
                    callback(new Error('error'))
                }else{
                    if(reg.test(value)){
                        callback();
                    }else{
                        callback(new Error('Length<%=rb.getString("MaoHao")%> 5~6 Digit <%=rb.getString("ZhengXing")%>'))
                    }
                } 
            },
            validateSectionTac = (rule,value,callback) => {

                if(value === ''){
                    callback(new Error('error'))
                }else{
                    if(vm.isNumeric(value)&&parseInt(value)>=1 && parseInt(value)<=16777215){
                        if(parseInt(value) == 16777214){
                            callback(new Error('error'))
                        }else{
                            callback();
                        }
                    }else{
                        callback(new Error('error'))
                    }
                } 
            },
            validateSectionSst = (rule,value,callback) => {
                var selectFlag = ['1--eMBB','2--URLLC','3--MIoT','4--V2X','5--HMTC'].includes(value); 
                if(selectFlag){
                    callback();
                }else{
                    if(value === ''){
                        callback(new Error('error'))
                    }else{
                        if(vm.isNumeric(value)&&parseInt(value)>=128 && parseInt(value)<=255){
                            callback();
                        }else{
                            callback(new Error('error'))
                        }
                    }
                    
                }
            },
            validateSectionSd = (rule,value,callback) => {

                if(value === ''){
                    callback()
                }else{
                    if(vm.isNumeric(value)&&parseInt(value)>=0 && parseInt(value)<=16777215){
                        callback();
                    }else{
                        callback(new Error('error'))
                    }
                } 
            },
            validateInterfaceIPorMask = (rule,value,callback) => {
                var regIpOrMask = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\/([0-9]|[12][0-9]|3[012])$/,
                    regIPV6OrMask = /^([\da-fA-F]{1,4}:){6}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^::([\da-fA-F]{1,4}:){0,4}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:):([\da-fA-F]{1,4}:){0,3}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:){2}:([\da-fA-F]{1,4}:){0,2}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:){3}:([\da-fA-F]{1,4}:){0,1}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:){4}:((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:){7}[\da-fA-F]{1,4}\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^:((:[\da-fA-F]{1,4}){1,6}|:)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^[\da-fA-F]{1,4}:((:[\da-fA-F]{1,4}){1,5}|:)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:){2}((:[\da-fA-F]{1,4}){1,4}|:)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:){3}((:[\da-fA-F]{1,4}){1,3}|:)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:){4}((:[\da-fA-F]{1,4}){1,2}|:)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:){5}:([\da-fA-F]{1,4})?\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:){6}:\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$/;
                if( value === '' || value === null || value === undefined) {
                    callback(new Error('error'))
                }else {
                    if(regIpOrMask.test(value) || regIPV6OrMask.test(value)){
                        callback();
                    }else{
                        callback(new Error('error'))
                    }
                    
                }
            },
            validateInterfaceMtu = (rule,value,callback) => {

                if(value === ''){
                    callback(new Error('error'))
                }else{
                    if(vm.isNumeric(value)&&parseInt(value)>=64 && parseInt(value)<=9220){
                        callback();
                    }else{
                        callback(new Error('error'))
                    }
                } 
            };
		return {
            egwCode:'',
            egwSn:'',
            activeName:'Basic',
            gnbSettingBasicForm:{
                plmn:'',
                ngAmfIp:'',
                ngAmfPort:'38412',
                upLinkIp:'',
                downLinkIp:'',
                GNB_idLength:32,
                AMF_idLength:22,
            },
            gnbBasicFormRules:{},

            gnbPlmnList:[],
            gnbNgAmfIpList:[],
            gnbUpLinkIpList:[],
            gnbDownLinkIpList:[],

            gnbIdMmeListTableData:[],
            delGnbIdMmeList:[],
            delGnbPlmnList:[],
            delNgAmfIpList:[],
            delGnbUplinkAddrList:[],
            delGnbDownlinkAddrList:[],
            gnbPlmnErrorMessage:'',
            ngAmfIpErrorMessage:'',
            gnbUpLinkIpErrorMessage:'',
            gnbDownLinkIpErrorMessage:'',
            oldGnbDataList:{
                basicInfo:{},
                enbListInfo:[]
            },
            idLengthList:[
                {label:'22',value:22,disabled:false},{label:'23',value:23,disabled:false},
                {label:'24',value:24,disabled:false},{label:'25',value:25,disabled:false},
                {label:'26',value:26,disabled:false},{label:'27',value:27,disabled:false},
                {label:'28',value:28,disabled:false},{label:'29',value:29,disabled:false},
                {label:'30',value:30,disabled:false},{label:'31',value:31,disabled:false},
                {label:'32',value:32,disabled:false}
            ],
            oldGnbConfigIdLengthForm:{
                GNB_idLength:32,
                AMF_idLength:22,
            },
            addSectionDialogShow:false,
            gnbSectionListTableUrl:'',
            addSectionForm:{
                plmn:'',
                tac:'',
                sst:'',
                sd:'',
            },
            addSectionRules:{
                plmn:[{required:true,validator: validateSectionPLMN}],
                tac:[{required:true,validator: validateSectionTac}],
                sst:[{required:true,validator: validateSectionSst}],
                sd:[{validator: validateSectionSd}]
            },
            gnbInterfaceListTableUrl:'',
            addInterfaceDialogShow:false,
            addInterfaceForm:{
                name:'',
                type:'N3a',
                ipAndMask:'',
                mtu:'1500',
                reassemblySwitch:'on'
            },
            addInterfaceRules:{
                name:[{required:true}],
                ipAndMask:[{required:true,validator: validateInterfaceIPorMask}],
                mtu:[{required:true,validator: validateInterfaceMtu}]
            },
            gnbInterfaceTotal:0,
            CardList:[],
            settingLinkSlideUrl:'',
			settingLinkSlideTitle:'',
			settingLinkSlideHeader:'',
			settingLinkSlideFooter:'',
			settingLinkSlidePosition:'',
			settingLinkSlideHeight:'',
			settingLinkSlideWidth:'',

		};
	},
	computed: {
        GNB_idLengthList(){
            var idLengthList = JSON.parse(JSON.stringify(this.idLengthList)),
                AMF_idLength = this.gnbSettingBasicForm.AMF_idLength;
            idLengthList.map((item)=>{
                if(item.value <= AMF_idLength){
                    item.disabled = true;
                }
            })
            return idLengthList
        },
        AMF_idLengthList(){
            var idLengthList = JSON.parse(JSON.stringify(this.idLengthList)),
                GNB_idLength = this.gnbSettingBasicForm.GNB_idLength;
            idLengthList.map((item)=>{
                if(item.value >= GNB_idLength){
                    item.disabled = true;
                }
            })
            return idLengthList
        },
        gnbInterfaceAddShow(){
            var nums = this.gnbInterfaceTotal;
            return nums < 8 ? true : false;
        },
    },
	methods: {
        // 初始化
		init(row,code,sn,status){
		    var vm =this;
			vm.egwCode = code;
            vm.egwSn = sn;
            vm.init5GParams();
		},
        init5GParams(){
            var vm = this,
                code = vm.egwCode,
                params={
                    egwCode:code
                };
            vm.gnbSectionListTableUrl = '${ctx}/egw/config/getGnbSliceList.action?egwCode='+code;
            axios.post('${ctx}/egw/config/getGnbConfig.action',stringify(params)).then(function(response){
                var data = response.data;
                if(data){
                    var basicInfo = data.basicInfo,
                        enbListInfo = data.enbListInfo;
                    
                    vm.gnbIdMmeListTableData = enbListInfo ? enbListInfo : [];
                    vm.CardList = basicInfo.CardList ? basicInfo.CardList : [];
                    vm.gnbSettingBasicForm.GNB_idLength = data.leftSize ? data.leftSize : '';
                    vm.gnbSettingBasicForm.AMF_idLength = data.rightSize ? data.rightSize : '';

                    vm.oldGnbConfigIdLengthForm.GNB_idLength = data.leftSize ? data.leftSize : '';
                    vm.oldGnbConfigIdLengthForm.AMF_idLength = data.rightSize ? data.rightSize : '';
                    if(basicInfo.HplmnList){
                        vm.gnbPlmnList = basicInfo.HplmnList;
                    }
                    if(basicInfo.IpList){
                        vm.gnbNgAmfIpList = basicInfo.IpList;
                    }
                    if(basicInfo.UplinkAddrList){
                        vm.gnbUpLinkIpList = basicInfo.UplinkAddrList;
                    }
                    if(basicInfo.DownlinkAddrList){
                        vm.gnbDownLinkIpList = basicInfo.DownlinkAddrList;
                    }
                    var initData = {
                        HplmnList:basicInfo.HplmnList||[],
                        IpList:basicInfo.IpList||[],
                        UplinkAddrList:basicInfo.UplinkAddrList||[],
                        DownlinkAddrList:basicInfo.DownlinkAddrList||[],
                    }
                    vm.oldGnbDataList.basicInfo = JSON.parse(JSON.stringify(initData));
                    vm.oldGnbDataList.enbListInfo = JSON.parse(JSON.stringify(enbListInfo));
                }
            }).catch(function(error){});
        },
        // Tab 切换
		tabClick(val){
            var vm = this,
                str = Math.random().toString();
            if(vm.activeName == 'PFM'){
                vm.gnbInterfaceListTableUrl = '${ctx}/egw/config/getGnbInterfaceList.action?egwCode='+vm.egwCode + '&randomCode=' + str;;
            }else{
                vm.init5GParams();
            }
        },
        // gnb  PLMN 添加事件
        addGnbPLMNs(){
            var vm = this,
                val = vm.gnbSettingBasicForm.plmn,
                reg = /^\d{5,6}$/,
                params={
                    Hplmn:vm.gnbSettingBasicForm.plmn
                },
                result = vm.gnbPlmnList.some(item=>item.Hplmn == val);
            
            if(val){
                if(reg.test(val)) {
                    if(result){
                        vm.gnbPlmnErrorMessage = '<%=rb.getString("YiCunZai")%>';
                    }else{
                        vm.gnbPlmnList.push(params);
                        vm.gnbPlmnErrorMessage = '';
                        vm.gnbSettingBasicForm.plmn = '';
                    }
                }else{
                    vm.gnbPlmnErrorMessage = '<%=rb.getString("PLMNFanWei")%>';
                }
            }
            
        },
        // gnb PLMN 删除事件
        gnbPlmnListDel(row,type){
            var vm = this;

            if(row.configIndex){
                vm.delGnbPlmnList.push(row.configIndex);
            }
            vm.gnbPlmnList = vm.gnbPlmnList.filter((items)=>{
                return items.Hplmn != row.Hplmn
            })
        },
        // gnb Ng-Amf Ip 新增事件
        addNgAmfIp(){
            var vm = this,
                val = vm.gnbSettingBasicForm.ngAmfIp,
                params={
                    Ip:vm.gnbSettingBasicForm.ngAmfIp
                };
            if(val){
                if(vm.isValidIP(val) || vm.isIPv6(val)) {
                    var result = vm.gnbNgAmfIpList.some(item=>item.Ip == val);
                    if(result){
                        vm.ngAmfIpErrorMessage = '<%=rb.getString("YiCunZai")%>';
                    }else{
                        vm.gnbNgAmfIpList.push(params);
                        vm.gnbSettingBasicForm.ngAmfIp = '';
                        vm.ngAmfIpErrorMessage = '';
                    }
                }else {
                    vm.ngAmfIpErrorMessage = 'Support configuration of IPV4 or IPV6';
                }
            }
        },
        // gnb Ng-Amf Ip 删除事件
        gnbNgAmfIpListDel(row){
            var vm = this;

            if(row.configIndex){
                vm.delNgAmfIpList.push(row.configIndex);
            }
            vm.gnbNgAmfIpList = vm.gnbNgAmfIpList.filter((items)=>{
                return items.Ip != row.Ip
            })
        },
        //gnb  UpLink IP添加事件
        addGnbUpLinkIp(){
            var vm = this,
                val = vm.gnbSettingBasicForm.upLinkIp,
                params={
                    N3bAddr:vm.gnbSettingBasicForm.upLinkIp
                };
            if(val){
                if(vm.isValidIP(val) || vm.isIPv6(val)) {
                    var result = vm.gnbUpLinkIpList.some(item=>item.N3bAddr == val);
                    if(result){
                        vm.gnbUpLinkIpErrorMessage = '<%=rb.getString("YiCunZai")%>';
                    }else{
                        vm.gnbUpLinkIpList.push(params);
                        vm.gnbSettingBasicForm.upLinkIp = '';
                        vm.gnbUpLinkIpErrorMessage = '';
                    }
                }else {
                    vm.gnbUpLinkIpErrorMessage = 'Support configuration of IPV4 or IPV6';
                }
            }
        },
        // gnb UpLink IP 删除事件
        gnbUpLinkIpListDel(row){
            var vm = this;
            if(row.configIndex){
                vm.delGnbUplinkAddrList.push(row.configIndex);
            }
            vm.gnbUpLinkIpList = vm.gnbUpLinkIpList.filter((items)=>{
                return items.N3bAddr != row.N3bAddr
            })
        },
        // gnb  downLink IP添加事件
        addGnbDownLinkIp(){
            var vm = this,
                val = vm.gnbSettingBasicForm.downLinkIp,
                params={
                    N3aAddr:vm.gnbSettingBasicForm.downLinkIp
                };

            if(val){
                if(vm.isValidIP(val) || vm.isIPv6(val)) {
                    var result = vm.gnbDownLinkIpList.some(item=>item.N3aAddr == val);
                    if(result){
                        vm.gnbDownLinkIpErrorMessage = '<%=rb.getString("YiCunZai")%>';
                    }else{
                        vm.gnbDownLinkIpList.push(params);
                        vm.gnbSettingBasicForm.downLinkIp = '';
                        vm.gnbDownLinkIpErrorMessage = '';
                    }
                }else {
                    vm.gnbDownLinkIpErrorMessage = 'Support configuration of IPV4 or IPV6';
                }
            }
        },
        //  downLink IP 删除事件
        gnbDownLinkIpListDel(row){
            var vm = this;
            if(row.configIndex){
                vm.delGnbDownlinkAddrList.push(row.configIndex);
            }
            vm.gnbDownLinkIpList = vm.gnbDownLinkIpList.filter((items)=>{
                return items.N3aAddr != row.N3aAddr
            })
        },
        // gnb 基本配置 基站配置 提交
        submitGnbBasicAndEnbList(){
            var vm = this;

            if(vm.activeName == 'Basic'){
                var errShow = false;
                if(vm.gnbPlmnList.length == 0){
                    vm.gnbPlmnErrorMessage = '<%=rb.getString("ZhiShaoTianJiaYiGe")%>';
                    errShow = true;
                }
                if(vm.gnbNgAmfIpList.length == 0){
                    vm.ngAmfIpErrorMessage = '<%=rb.getString("ZhiShaoTianJiaYiGe")%>';
                    errShow = true;
                }
                if(vm.gnbUpLinkIpList.length == 0){
                    vm.gnbUpLinkIpErrorMessage = '<%=rb.getString("ZhiShaoTianJiaYiGe")%>';
                    errShow = true;
                }
                if(vm.gnbDownLinkIpList.length == 0){
                    vm.gnbDownLinkIpErrorMessage = '<%=rb.getString("ZhiShaoTianJiaYiGe")%>';
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
                vm.gnbPlmnList.map((item,index)=>{
                    if(!item.configIndex){
                        addHplmnList.push(item)
                    }
                })
                vm.gnbNgAmfIpList.map((item,index)=>{
                    if(!item.configIndex){
                        addIpList.push(item)
                    }
                })
                vm.gnbUpLinkIpList.map((item,index)=>{
                    if(!item.configIndex){
                        addUplinkAddrList.push(item)
                    }
                })
                vm.gnbDownLinkIpList.map((item,index)=>{
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
                params.delPlmnList = vm.delGnbPlmnList.join(',');
                params.delIpList = vm.delNgAmfIpList.join(',');
                params.delUplinkAddrList = vm.delGnbUplinkAddrList.join(',');
                params.delDownlinkAddrList = vm.delGnbDownlinkAddrList.join(',');

                if(vm.gnbSettingBasicForm.GNB_idLength != vm.oldGnbConfigIdLengthForm.GNB_idLength){
                    params.leftSize = vm.gnbSettingBasicForm.GNB_idLength
                }
                if(vm.gnbSettingBasicForm.AMF_idLength != vm.oldGnbConfigIdLengthForm.AMF_idLength){
                    params.rightSize = vm.gnbSettingBasicForm.AMF_idLength
                }
            }else{
                vm.gnbIdMmeListTableData.map((item,index)=>{
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
                params.delEnbList = vm.delGnbIdMmeList.join(',');
            }
            params.enbListInfo = JSON.stringify(enbListInfo);
            axios.post("${ctx}/egw/config/setGnbConfig.action",stringify(params)).then(function(response){
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
                eventBus.$emit("link-init",'add','','gnb');
            });
        },
        // 修改 链路总配置
        editGnbIdMme(row,event){
            var vm = this;

            vm.settingLinkSlideHeader = true;
            vm.settingLinkSlideUrl = '${ctx}/egw/pageForward/goEGWSettingsEnbLink.action';
            vm.settingLinkSlideFooter = true;
            vm.settingLinkSlidePosition = 'top';
            vm.settingLinkSlideHeight = '100%';
            vm.settingLinkSlideWidth = '100%';
            vm.settingLinkSlideTitle = 'edit';
            vm.$refs.egwSettingLinkSlide.showSlide(()=>{
                eventBus.$emit("link-init",'edit',row,'gnb');
            });
        },
        // 删除 gnb 链路总配置
        delGnbIdMme(row){
            var vm = this;
            if(row.configIndex){
                vm.delGnbIdMmeList.push(row.configIndex);
            }
            vm.gnbIdMmeListTableData = vm.gnbIdMmeListTableData.filter((items)=>{
                return (items.enodebId +''+ items.Hplmn) != (row.enodebId +''+ row.Hplmn)
            })
        },
        settingLinkSubmit(){
			var vm = this;
			eventBus.$emit("egw-settingLink-ok");
		},
        // 链路设置保存 更改enbList表格信息
        editGnbList(data,type){
            var vm = this,
                editDataList = data,
                isExist = false;
            vm.gnbIdMmeListTableData.forEach((items,index,array)=>{
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
                vm.gnbIdMmeListTableData.unshift(data);
            }
        },
		// 关闭链路配置置页面
		closeSettingLinkSlide(){
			var vm = this;
			vm.$refs.egwSettingLinkSlide.hide();
		},
        // 切片配置 打开新增弹窗
        sectionAdd(){
            var vm = this;
            vm.addSectionDialogShow = true;
        },
        // 切片配置删除 
        delSection(row){
            var vm = this,
                urls="${ctx}/egw/config/setGnbConfig.action",
                params={
                    egwCode:vm.egwCode,
                },
                delOtherConfig ={
                    sliceList:row.index
                };
            params.delOtherConfig = JSON.stringify(delOtherConfig);
            var delTips = '<%=rb.getString("QiePianShanChuTiShi")%>';
            var confirmHint ='<div style="font-size:14px;color:#333333">'+ '<%=rb.getString("QueDingShanChuRenWu")%>' +'</div>'+'<div style="font-size:12px;color:#999999">'+ delTips +'</div>';
            vm.$confirm(confirmHint,'<%=rb.getString("QueRen")%>',{
                customClass:'warningConfirm',
                confirmButtonText:'<%=rb.getString("QueDing")%>',
                cancalButtonText:'<%=rb.getString("QuXiao")%>',
                dangerouslyUseHTMLString:true
            }).then(()=>{
                axios.post(urls,stringify(params)).then(res=>{
                    var data = res.data;
                    if(data["success"]){
                        vm.$message({
                            message: '<%=rb.getString("MingLingYiXiaFa")%>',
                            type:'success',
                        });
                        vm.$refs.gnbSectionListTable.refresh();
                    }else{
                        vm.$message.error(data["message"])
                    }
                })
            }).catch(()=>{})
        },
        // 切片配置新增提交
        addSectionSubmit(){
            var vm = this,
                urls="${ctx}/egw/config/setGnbConfig.action",
                params={
                    egwCode:vm.egwCode,
                },
                otherConfig ={
                    sliceList:[]
                };
            var sectionItem = Object.assign({},vm.addSectionForm)
            if(!sectionItem.sd){
                delete sectionItem.sd
            }
            otherConfig.sliceList.push(sectionItem);
            params.otherConfig = JSON.stringify(otherConfig);
            vm.$refs["addSectionForm"].validate( valid => {
                if(valid){
                    axios.post(urls,stringify(params)).then(res=>{
                        var data = res.data;
                        if(data["success"]){
                            vm.$message({
                                message: '<%=rb.getString("MingLingYiXiaFa")%>',
                                type:'success',
                            });
                            vm.closeAddSection();
                            vm.$refs.gnbSectionListTable.refresh();
                        }else{
                            vm.$message.error(data["message"])
                        }
                    })
                }else{
                    return false
                }
            })
        },
        // 切片配置新增取消
        closeAddSection(){
            var vm = this,
                params={
                    plmn:'',
                    tac:'',
                    sst:'',
                    sd:'',
                };
            vm.addSectionDialogShow = false;
            Object.assign(vm.addSectionForm,params);
            vm.$refs.addSectionForm.clearValidate();
        },
        // PMF 设备连接 打开新增弹窗
        interfaceAdd(){
            var vm = this;
            vm.addInterfaceDialogShow = true;
        },
        // PMF 设备连接 删除事件
        delInterface(row){
            var vm = this,
                urls="${ctx}/egw/config/setGnbConfig.action",
                params={
                    egwCode:vm.egwCode,
                },
                delOtherConfig ={
                    interfaceList:row.index
                };
            params.delOtherConfig = JSON.stringify(delOtherConfig);
            vm.$confirm('<%=rb.getString("QueDingShanChuRenWu")%>','<%=rb.getString("QueRen")%>',{
                customClass:'warningConfirm',
                confirmButtonText:'<%=rb.getString("QueDing")%>',
                cancalButtonText:'<%=rb.getString("QuXiao")%>',
                dangerouslyUseHTMLString:true
            }).then(()=>{
                axios.post(urls,stringify(params)).then(res=>{
                    var data = res.data;
                    if(data["success"]){
                        vm.$message({
                            message: '<%=rb.getString("MingLingYiXiaFa")%>',
                            type:'success',
                        });
                        vm.$refs.gnbInterfaceListTable.refresh();
                    }else{
                        vm.$message.error(data["message"])
                    }
                })
            }).catch(()=>{})
        },
        // PMF 设备连接 新增提交
        addInterfaceSubmit(){
            var vm = this,
                urls="${ctx}/egw/config/setGnbConfig.action",
                params={
                    egwCode:vm.egwCode,
                },
                otherConfig ={
                    interfaceList:[]
                };
            otherConfig.interfaceList.push(vm.addInterfaceForm);
            params.otherConfig = JSON.stringify(otherConfig);
            vm.$refs["addInterfaceForm"].validate( valid => {
                if(valid){
                    axios.post(urls,stringify(params)).then(res=>{
                        var data = res.data;
                        if(data["success"]){
                            vm.$message({
                                message: '<%=rb.getString("MingLingYiXiaFa")%>',
                                type:'success',
                            });
                            vm.closeAddInterface();
                            vm.$refs.gnbInterfaceListTable.refresh();
                        }else{
                            vm.$message.error(data["message"])
                        }
                    })
                }else{
                    return false
                }
            })
        },
        // PMF 设备连接 新增取消
        closeAddInterface(){
            var vm = this,
                params={
                    name:'',
                    type:'N3a',
                    ipAndMask:'',
                    mtu:'',
                    reassemblySwitch:'on'
                };
            vm.addInterfaceDialogShow = false;
            Object.assign(vm.addInterfaceForm,params);
            vm.$refs.addInterfaceForm.clearValidate();
        },
        // 验证输入的是否是数字
        isNumeric(str) {
            if(str.length==0){
                return false;
            }
            for(var i=0;i<str.length;i++){
                if(str.charAt(i)<"0" || str.charAt(i)>"9"){
                    return false;
                }
            }
            return true;  
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
        // gnb Interface
        gnbInterfaceLoadSuccess(data){
            var vm = this;
            if(data){
                vm.gnbInterfaceTotal = data.total
            }
        },
        // 同步
        syncSubmit(){
            var vm = this,
                urls='${ctx}/egw/config/refreshConfig.action',
                params={
                    egwCode: vm.egwCode,
                    refreshType: 'SIGNALINGGATEWAY5G'
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
        eventBus.$off('edit-enbList').$on('edit-enbList',this.editGnbList);
        eventBus.$off('close-linkSetting').$on('close-linkSetting',this.closeSettingLinkSlide);
	}
});

</script>
