<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
#egwSecurityGatewayPage{
    width:100%;
    height:calc(100% - 10px);
    position: relative;
}
#egwSecurityGatewayPage .itemMainBoxCls{
	border:1px solid #d5dcec;
	border-radius:10px;
	margin:8px;
	background:#fff;
	height:100%;
}
#egwSecurityGatewayPage .itemMainBoxCls .el-tab-pane{
	position: relative;
}
#egwSecurityGatewayPage .itemMainBoxTitle {
	height:36px;
    line-height: 36px;
	font-size:14px;
	font-weight:bold;
    position: relative;
    margin-bottom: 10px;
}
#egwSecurityGatewayPage .egwTabPaneItemBoxCls{
	flex: 1;
}
#egwSecurityGatewayPage .egwTabPaneTableBoxCls{
    height:calc(100% - 60px);
    overflow: hidden;
    box-sizing: border-box;
    border:1px solid #d5dcec;
    border-radius: 4px;
    margin: 0px 50px 0px 20px;
    position: relative;
}
#egwSecurityGatewayPage .toolbarBoxCls{
    padding: 10px 0px;
}
#egwSecurityGatewayPage .egwTabPaneContent{
    margin: 25px 0px 0px 40px;
    display: flex;
    flex-direction: column;
    height: 100%;
    overflow: auto;
}
#egwSecurityGatewayPage .egwTabPaneContent .egwTabPaneItemCls{
    margin-bottom: 20px;
    font-size: 14px;
}
#egwSecurityGatewayPage .itemMainBoxFooter{
    display: flex;
    align-items: center;
    border-top : 1px solid #E9E9E9;
	height:48px;
    background-color: #fff;
    box-sizing: border-box;
    width: calc(100% - 0px);
    position: absolute;
    bottom: 0px;
}
.egwSettingAddDialogCls .el-form-item,.cmpServerBoxCls .el-form-item{
    display: inline-block;
    width: 48%;
}
.egwSettingAddDialogCls .el-form-item__error,.cmpServerBoxCls .el-form-item__error{
    padding-top: 0px;
    top:30px!important;
}
.egwSettingAddDialogCls .addIPsecConfigContentCls{
    height: 50%;
    overflow: auto;
}
.egwSettingAddDialogCls  .addIPsecConfigItemBoxCls{
    width: 100%;
    padding-left: 20px;
}
.egwSettingAddDialogCls .addIPsecConfigItemTitleCls{
    display:flex;
    align-items:center;
    width: 100%;
    margin-bottom:30px;
    font-size: 14px;
    font-weight: 500;
}
.egwSettingAddDialogCls .dotCls{
    height: 6px;
    width: 6px;
    background-color: #333333;
    border-radius: 3px;
    margin-right: 10px;
}
.egwSettingAddDialogCls .bottomLine,#egwSecurityGatewayPage .bottomLine{
    background-color:#E9E9E9;
    width: 100%;
    height: 1px;
    margin-bottom: 20px; 
}
#egwSecurityGatewayPage .lastFetchInfoBoxCls{
    position: relative;
    height: 70px;
    margin:0px 20px 20px 0px;;
    padding: 15px 20px;
    border: 1px dashed #D5DCEC;
    border-radius: 4px;
    background-color: rgba(213, 220, 236, 0.1);
}
#egwSecurityGatewayPage .lastFetchInfoItem{
    margin-bottom: 5px;
    width: 40%;
    display: flex;
}
#egwSecurityGatewayPage .lastFetchInfoBoxCls >div:first-child{
    color: #7A7992;
    margin-bottom: 15px;
    font-weight: 550;
}
#egwSecurityGatewayPage .FetchSuccessIcon::before{
    color: #67D972;
    font-size: 15px;
}
#egwSecurityGatewayPage .FetchFailedIcon::before{
    color: #E88282;
    font-size: 15px;
}
#egwSecurityGatewayPage .UnSyncStatus::before, .egwSettingAddDialogCls .UnSyncStatus::before{
    font-size: 18px;
    color:#D8C3D9;
}
#egwSecurityGatewayPage .fetchLabelEnCls{
    width:150px; 
}
#egwSecurityGatewayPage .fetchLabelZhCls{
    width:110px; 
}
#egwSecurityGatewayPage .SyncStatus::before,.egwSettingAddDialogCls .SyncStatus::before{
    font-size: 18px;
    color:#4ED76E;
}
#egwSettingPage .beyondEllipsisCls,.egwSettingAddDialogCls .beyondEllipsisCls{
    width: 140px;
    overflow:hidden;
    white-space:nowrap;
    text-overflow:ellipsis;
}
#egwSecurityGatewayPage .IPsecConfigEnableClickBox{
    height: 20px;
    width: 50px;
    position: absolute;
    top:0px;
    left: 0px;
    right: 0px;
    bottom: 0px;
    margin: auto;
    z-index: 66;
    cursor: pointer;
    opacity: 0;
}
#egwSecurityGatewayPage .formItemItemBoxCls{
    margin-left: 20px;
}
#egwSecurityGatewayPage .applyCertBoxCls{
    border: 1px dashed #D5DCEC;
    border-radius: 4px;
    background-color: rgba(213, 220, 236, 0.1);
    padding: 20px;
    margin:0px 40px 20px 0px;
}
#egwSecurityGatewayPage .applyCertBoxCls .circleIconBoxCls{
    height: 25px;
    width: 25px;
    display: flex;
    justify-content: center;
    align-items: center;
    border-radius: 12px;
    background-color: #E4F1FF;
    margin-right: 5px;
}
#egwSecurityGatewayPage .applyCertBoxCls >div:first-child{
    font-size: 14px;
    color: #000;
    margin-bottom: 20px;
    font-weight: 550;
    display: flex;
    align-items: center;
}
#egwSecurityGatewayPage .certificateStatusItemBoxCls{
    min-width: 300px;
    display: flex;
    margin-left: 20px;
}
#egwSecurityGatewayPage .certificateStatusItemBoxCls >div{
    font-size: 12px;
    color: #333333;
    margin-bottom: 10px;
    white-space: nowrap;
    width: 40%;
}
#egwSecurityGatewayPage .certificateStatusItemBoxCls >div span{
    display: inline-block;
    width: 140px;
    color:#666666;
}
#egwSecurityGatewayPage .inputRightBoxCls{
    display: inline-block;
    border: 1px solid #DFE2EE;
    border-left:none; 
    border-radius: 4px;
    height: 26px;
    width: 26px;
    text-align: center;
    box-sizing: border-box;
    position: relative;
    left:-3px;
    top: -1px;
}
#egwSecurityGatewayPage .inputRightBoxCls .opBtnBoxCls{
    position: relative;
    top: 3px;
}
.egwSettingAddDialogCls .certTableBoxCls{
    height: 300px;
}
.egwSettingAddDialogCls .labelSlotCls>span{
    color: #999999;
    font-size: 12px;
    margin-left: 10px;
}
.egwSettingAddDialogCls .el-input__suffix{
    height: 26px;
    display: flex;
    align-items: center;
}
.egwSettingAddDialogCls .el-upload__tip{
    color:red;
    margin-top: unset;
}
.egwSettingAddDialogCls .tooltipCls.is-dark{
    background : #959595 ;
    color : #FFFFFF ;
}
.egwSettingAddDialogCls .tooltipCls[x-placement^=top] .popper__arrow ,
.egwSettingAddDialogCls .tooltipCls[x-placement^=top] .popper__arrow::after{
    border-top-color: #959595!important;
}

.egwSettingAddDialogCls .tooltipCls[x-placement^=bottom] .popper__arrow ,
.egwSettingAddDialogCls .tooltipCls[x-placement^=bottom] .popper__arrow::after {
    border-bottom-color: #959595!important;
}
.egwSettingAddDialogCls .tooltipCls[x-placement^=right] .popper__arrow ,
.egwSettingAddDialogCls .tooltipCls[x-placement^=right] .popper__arrow::after {
    border-right-color: #959595!important;
}
.egwSettingAddDialogCls .tooltipCls[x-placement^=left] .popper__arrow ,
.egwSettingAddDialogCls .tooltipCls[x-placement^=left] .popper__arrow::after {
    border-left-color: #959595!important;
}
</style>

<div id="egwSecurityGatewayPage">
    <div class="newIconBoxCls-bt" style="right:20px;top:5px;" @click="syncSubmit" tip='<%=rb.getString("TongBu")%>'>
        <span class="el-icon el-icon-circle-refresh"></span>
    </div>	
	<div class="itemMainBoxCls">
		<el-tabs class="fit newTabs" v-model="activeName" style='height:100%' @tab-click="tabClick">
            <el-tab-pane label="<%=rb.getString("IPsecPeiZhi")%>" name="IPsec">
                <div class="egwTabPaneContent">
                    <div class="itemMainBoxTitle">
                        <span class="el-icon el-icon-splitGroup"></span>
                        Security Group
                        <span class="el-icon el-icon-circle-add" @click="addIPsecConfigClick" style="position:absolute;right:55px;"></span>
                    </div>
                    <div class="egwTabPaneTableBoxCls">
                        <el-ctable
                            id="IPsecConfigTable"
                            :url="IPsecConfigTableUrl"
                            :query-params="queryIPsecConfigParams" 
                            ref="IPsecConfigTable" 
                            :time="6"
                            :height="'100%'" 
                            pagination="true"
                         >
                                <!-- 列表toolbar -->
                            <template slot="toolbar">
                                <div class="toolbarBoxCls">
                                    <el-query type="normal" @query="queryIPsecConfig" placeholder="Group Name"></el-query>
                                </div>
                            </template>
                                <!-- 列表columns -->
                            <el-table-column label="" width="100" class-name="no-text-tips">
                                <template slot-scope="scope"><!-- 将元素或组件表示为作用域插槽          -->
                                    <div class="el-icon el-icon-operation-edit" @click="IPsecConfigModify(scope.row,event)" style="cursor: pointer;"></div>
                                    <div class="el-icon el-icon-operation-delete" @click="IPsecConfigDel(scope.row,event)" style="cursor: pointer;"></div>
                                </template>
                            </el-table-column>
                            <el-table-column prop="" show-overflow-tooltip label="Enable" width="100">
                                <template slot-scope="scope"><!-- 将元素或组件表示为作用域插槽          -->
                                    <div style="position:relative;">
                                        <el-switch v-model="scope.row.switch_enable" style="height: 18px;margin-left: 10px;"
                                            active-color="#4D84FF"
                                            active-value="1"
                                            inactive-value="0">	
                                        </el-switch>
                                        <div class="IPsecConfigEnableClickBox" @click="IPsecConfigEnableChange(scope.row)"></div>
                                    </div>
                                </template>
                            </el-table-column>
                            <el-table-column prop="group_name" show-overflow-tooltip label="Group Name" min-width="240" ></el-table-column>
                        </el-ctable>
                    </div>
                </div>
            </el-tab-pane>
            <el-tab-pane label="Certificate Installation" name="Cert">
                <div class="egwTabPaneContent" id="IPsecCertBox" style="height:calc(100% - 60px);">
                    <div class="lastFetchInfoBoxCls">
                        <div><%=rb.getString("CongWCGHuoQuZhengShuXinXi")%></div>
                        <div style="display: flex;">
                            <div class="lastFetchInfoItem">
                                <div :class="fetchLabelClass"><%=rb.getString("ShangCiHuoQuShiJian")%><%=rb.getString("MaoHao")%></div>
                                <div>{{IPsecCertForm.refresh_time}}</div>
                            </div>
                            <div class="lastFetchInfoItem">
                                <div :class="fetchLabelClass"><%=rb.getString("ShangCiHuoQuZhuangTai")%><%=rb.getString("MaoHao")%></div>
                                <div v-if="IPsecCertForm.refresh_status == '1'">
                                    <span class="el-icon el-icon-circle-success FetchSuccessIcon"></span>
                                    <span><%=rb.getString("HuoQuChengGong")%></span>
                                </div>
                                <div v-if="IPsecCertForm.refresh_status == '0'">
                                    <span class="el-icon el-icon-circle-close FetchFailedIcon"></span>
                                    <span><%=rb.getString("HuoQuShiBai")%></span>
                                </div>
                            </div>
                        </div>
                        <div class="newIconBoxCls-bt" style="right:20px;top:35px;" @click="refreshCertInfo">		
							<span class=" el-icon-circle-refresh el-icon"></span>
						</div>
                    </div>
                    <div class="cmpServerBoxCls">
                        <el-form label-position="top" ref="IPsecCertForm" :model='IPsecCertForm' :rules='IPsecCertRules'>
                            <div class="itemMainBoxTitle">
                                <span class="el-icon el-icon-splitGroup"></span>
                                CMP Server
                            </div>
                            <div class="formItemItemBoxCls">
                                <el-form-item label="Enable" label-width="110px" prop="cert_manage" class="selectWidthCls">
                                    <el-select v-model='IPsecCertForm.cert_manage'>
                                        <el-option label='Enable' value='1'></el-option>
                                        <el-option label='Disable' value='0'></el-option>
                                    </el-select>
                                </el-form-item>
                                <el-form-item label="CMP Server Address" label-width="110px" prop="cmp_server_address">
                                    <el-input v-model.trim='IPsecCertForm.cmp_server_address' placeholder="String,Length：0~512"></el-input>
                                </el-form-item>
                                <el-form-item label="Region" label-width="110px" prop="country">
                                    <el-input v-model.trim='IPsecCertForm.country'></el-input>
                                </el-form-item>
                                <div style="margin-bottom:20px;">
                                    <el-button type="primary"  @click="saveCertInfo"><%=rb.getString("QueDing")%></el-button>
                                    <el-button @click="closeLinkSetting"><%=rb.getString("QuXiao")%></el-button>
                                </div> 
                            </div>
                            <div class="bottomLine"></div>
                            <div class="itemMainBoxTitle">
                                <span class="el-icon el-icon-splitGroup"></span>
                                <%=rb.getString("ZhengShuGengXin")%>
                            </div>
                            <div class="applyCertBoxCls">
								<div>
									<div class="circleIconBoxCls">
										<span class="el-icon el-icon-circle-cert greyIcon"></span>
									</div>
									Apply Certificate Status
								</div>
								<div>
									<div class="certificateStatusItemBoxCls">
										<div>
											<span>Certificate Name</span>
											{{IPsecCertForm.file_name}}
										</div>
									</div>
									<div class="certificateStatusItemBoxCls">
										<div>
											<span>Certificate Start Time</span>
											{{IPsecCertForm.valid_start_time}}
										</div>
										<div>
											<span>Certificate End Time</span>
											{{IPsecCertForm.valid_end_time}}
										</div>
									</div>
								</div>	
							</div>
                            <div class="formItemItemBoxCls">
                                <el-form-item label="<%=rb.getString("CAZhengShu")%>" label-width="110px" prop="ca_cert" class="selectWidthCls">
									<el-select v-model='IPsecCertForm.ca_cert'>
										<el-option v-for="item in ca_certList" :label='item.file_name' :value='item.file_name'></el-option>
									</el-select>
									<el-tooltip popper-class="tooltipCls" content='<%=rb.getString("CAZhengShu")%>' placement='bottom'>
										<div class="inputRightBoxCls">
											<div class="opBtnBoxCls" @click="viewCertClick('CA')"><span class="el-icon el-icon-operation-result greyIcon"></span></div>
										</div>
									</el-tooltip>
								</el-form-item>
                                <el-form-item label="IPsec Certificate" label-width="110px" prop="ipsec_cert" class="selectWidthCls">
									<el-select v-model='IPsecCertForm.ipsec_cert' @change="ipsecCertChange">
										<el-option v-for="item in ipsec_certList" :label='item.file_name' :value='item.file_name'></el-option>
									</el-select>
									<el-tooltip popper-class="tooltipCls" content='IPsec Certificate' placement='bottom'>
										<div class="inputRightBoxCls">
											<div class="opBtnBoxCls" @click="viewCertClick('IPsec')"><span class="el-icon el-icon-operation-result greyIcon"></span></div>
										</div>
									</el-tooltip>
								</el-form-item>
								<el-form-item label="IPsec Private Key" label-width="110px" prop="private_key" class="selectWidthCls">
									<el-select v-model='IPsecCertForm.private_key'>
										<el-option v-for="item in private_keyList" :label='item.file_name' :value='item.file_name'></el-option>
									</el-select>
									<el-tooltip popper-class="tooltipCls" content='IPsec Private Key' placement='bottom'>
										<div class="inputRightBoxCls">
											<div class="opBtnBoxCls" @click="viewCertClick('Key')"><span class="el-icon el-icon-operation-result greyIcon"></span></div>
										</div>
									</el-tooltip>
								</el-form-item>
                            </div>
                        </el-form>
                    </div>
                </div>
                <div class='itemMainBoxFooter'>
                    <el-button type="primary" @click="issueCertFile" style="margin-left:20px;">Send to WCG</el-button>
                    <el-button @click="closeLinkSetting"><%=rb.getString("QuXiao")%></el-button>
                    <el-button type="primary" @click="updateCertFile" :disabled="issueAndUpdateCertBtnDis"><%=rb.getString("ZhengShuGengXin")%></el-button>
                </div>
            </el-tab-pane>
            <el-tab-pane label="EAP-AKA Authentication Service" name="User">
                <div class="egwTabPaneContent">
                    <div class="egwTabPaneItemBoxCls">
                        <div class="itemMainBoxTitle">
                            <span class="el-icon el-icon-splitGroup"></span>
                            <%=rb.getString("RenZhengYongHu")%>
                            <span class="el-icon el-icon-circle-add" style="position:absolute;right:130px;" @click="addVerifiedAccountClick"></span>
                            <span class="el-icon el-icon-operation-import" style="position:absolute;right:105px;top:9px;" @click="verifiedAccountImportClick('user')"></span>
                            <span class="el-icon el-icon-operation-export" style="position:absolute;right:80px;top:9px;" @click="exportVerifiedAccount"></span>
                            <span class="el-icon el-icon-operation-blacklist" style="position:absolute;right:55px;top:9px;" @click="blockListClick"></span>
                        </div>
                        <div class="egwTabPaneTableBoxCls">
                            <el-ctable
                                id="verifiedAccountTable"
                                :url="verifiedAccountTableUrl"
                                :query-params="queryVerifiedAccountParams" 
                                ref="verifiedAccountTable" 
                                :time="6"
                                :height="'100%'" 
                                @selection-change='verifiedAccountSelect'
                                pagination="true">
                                    <!-- 列表toolbar -->
                                <template slot="toolbar">
                                    <div class="toolbarBoxCls">
                                       <el-query type="normal" @query="queryVerifiedAccount" placeholder="<%=rb.getString("IMSI")%>"></el-query>
                                    </div>
                                </template>
                                    <!-- 列表columns -->
                                <el-table-column type="selection" width="45"></el-table-column>
                                <el-table-column label="" width="30" class-name="no-text-tips">
                                    <template slot-scope="scope"><!-- 将元素或组件表示为作用域插槽          -->
                                        <div class="el-icon el-icon-operation-more" @click="verifiedAccountOptClick(scope.row,event)" v-clickoutside="hideverifiedAccountMenus"></div>
                                    </template>
                                </el-table-column>
                                <el-table-column prop="imsi" show-overflow-tooltip label="IMSI" min-width="130"></el-table-column>
                                <el-table-column prop="key" show-overflow-tooltip label="KEY" min-width="240" ></el-table-column>
                                <el-table-column prop="opc" show-overflow-tooltip label="OPC" min-width="240" ></el-table-column>
                                <el-table-column prop="userSn" show-overflow-tooltip label="Sn" min-width="130"></el-table-column>
                                <el-table-column prop="userMac" show-overflow-tooltip label="Mac" min-width="130" ></el-table-column>
                                <el-table-column prop="rcConstId" show-overflow-tooltip label="Rc Const Id" min-width="130" ></el-table-column>
                                <el-table-column prop="vendor" show-overflow-tooltip label="Vendor" min-width="130" ></el-table-column>
                            </el-ctable>
                            <el-cmenu ref="verifiedAccountMenu" :data="verifiedAccountMenus" @click="verifiedAccountMenuClick"></el-cmenu>
                            <!-- 批量操作  -->
                            <el-bulk target="verifiedAccountTable" :list="verifiedAccountSelection" :row-key="'imsi'" show-prop="imsi"
                                :message="bulkTableMessage">
                                <template slot="button">
                                    <a class="linkbutton" @click="verifiedAccountDel('batch')"><span><%=rb.getString("ShanChu")%></span></a>
                                    <a class="linkbutton" @click="verifiedAccountMove('batch')"><span><%=rb.getString("YiDongDaoHeiMingDan")%></span></a>
                                </template>
                            </el-bulk>
                        </div>
                    </div>
                </div>
            </el-tab-pane>
		</el-tabs>
	</div>
    <!-- IPsec配置新增 弹窗 -->
	<el-dialog id="addIPsec" class="egwSettingAddDialogCls gnbConfigAddDialog" 
        top="15vh" 
        title='<%=rb.getString("TianJia")%>'
        width="60%"
        :visible.sync="addIPsecConfigShow" 
        :close-on-click-modal="false" 
        append-to-body
        @close="closeAddIPsecConfig"
     >		
        <div class="addIPsecConfigContentCls">
		    <el-form label-position="top" ref="addIPsecConfigForm" :model='addIPsecConfigForm' :rules='addIPsecConfigRules'>     
                <div class="addIPsecConfigItemTitleCls">
                    <div class="dotCls"></div>
                    <div><%=rb.getString("JiBenSheZhi")%></div>
                </div>
                <div class="addIPsecConfigItemBoxCls">
                    <el-form-item label="Group Name" style="min-width:400px;" prop="group_name" >
                        <el-input maxlength="64" v-model.trim='addIPsecConfigForm.group_name' :disabled="addIPsecConfigType == 'edit'"></el-input>
                    </el-form-item>
                    <el-form-item label="Enable" style="min-width:400px;" prop="switch_enable">
                        <el-select v-model='addIPsecConfigForm.switch_enable'>
                            <el-option label='Enable' value='1'></el-option>
                            <el-option label='Disable' value='0'></el-option>
                        </el-select>
                    </el-form-item>
                    <el-form-item v-show="secretKeyShow" label="Secret Key" style="min-width:400px;" prop="secret_key" class='validate-item'>
                        <el-input maxlength="64" v-model.trim='addIPsecConfigForm.secret_key'>
                            <template slot="append"><%=rb.getString("FanWei")%>:0~64 Digit string</template>
                        </el-input>
                    </el-form-item>
                    <el-form-item label="Fragmentation" style="min-width:400px;" prop="fragmentation">
                        <el-select v-model='addIPsecConfigForm.fragmentation'>
                            <el-option label='YES' value='yes'></el-option>
                            <el-option label='ACCEPT' value='accept'></el-option>
                            <el-option label='FORCE' value='force'></el-option>
                            <el-option label='NO' value='no'></el-option>
                        </el-select>
                    </el-form-item>
                </div>
                <div class="bottomLine"></div>
                <div class="addIPsecConfigItemBoxCls">
                    <el-form-item label="Left" style="min-width:400px;" prop="left_ip" class='validate-item'>
                        <el-input v-model.trim='addIPsecConfigForm.left_ip' maxlength="64">
                            <template slot="append"><%=rb.getString("FanWei")%>:0~64 Digit string</template>
                        </el-input>
                    </el-form-item>
                    <el-form-item label="Left ID" style="min-width:400px;" prop="left_id" class='validate-item'>
                        <el-input maxlength="64" v-model.trim='addIPsecConfigForm.left_id'>
                            <template slot="append"><%=rb.getString("FanWei")%>:0~64 Digit string</template>
                        </el-input>
                    </el-form-item>
                    <el-form-item label="Left Auth" style="min-width:400px;" prop="left_auth">
                        <el-select v-model='addIPsecConfigForm.left_auth'>
                            <el-option label='psk' value='psk'></el-option>
                            <el-option label='pubkey' value='pubkey'></el-option>
                            <el-option label='eap-aka' value='eap-aka'></el-option>
                        </el-select>
                    </el-form-item>
                    <el-form-item label="Left Source IP" style="min-width:400px;" prop="left_source" class='validate-item'>
                        <el-input v-model.trim='addIPsecConfigForm.left_source' maxlength="64">
                            <template slot="append"><%=rb.getString("FanWei")%>:0~64 Digit string,Support IPv4 or IPv6</template>
                        </el-input>
                    </el-form-item>
                    <el-form-item label="Left Subnet" style="min-width:400px;" prop="left_subnet" class='validate-item'>
                        <el-input v-model.trim='addIPsecConfigForm.left_subnet' maxlength="64">
                            <template slot="append"><%=rb.getString("FanWei")%>:0~64 Digit string</template>
                        </el-input>
                    </el-form-item>
                    <el-form-item v-show="leftCertShow" label="Left Cert" style="min-width:400px;" prop="left_cert">
                        <el-select v-model='addIPsecConfigForm.left_cert'>
                            <el-option v-for="item in addIPsecConfigCertData" :label='item.file_name' :value='item.file_name'></el-option>
                        </el-select>
                    </el-form-item>
                </div>
                <div class="bottomLine"></div>
                <div class="addIPsecConfigItemBoxCls">
                    <el-form-item label="Right" style="min-width:400px;" prop="right_ip" class='validate-item'>
                        <el-input v-model.trim='addIPsecConfigForm.right_ip' maxlength="64">
                            <template slot="append"><%=rb.getString("FanWei")%>:0~64 Digit string</template>
                        </el-input>
                    </el-form-item>
                    <el-form-item label="Right ID" style="min-width:400px;" prop="right_id" class='validate-item'>
                        <el-input maxlength="64" v-model.trim='addIPsecConfigForm.right_id'>
                            <template slot="append"><%=rb.getString("FanWei")%>:0~64 Digit string</template>
                        </el-input>
                    </el-form-item>
                    <el-form-item label="Right Auth" style="min-width:400px;" prop="right_auth">
                        <el-select v-model='addIPsecConfigForm.right_auth'>
                            <el-option label='psk' value='psk'></el-option>
                            <el-option label='pubkey' value='pubkey'></el-option>
                            <el-option label='eap-aka' value='eap-aka'></el-option>
                        </el-select>
                    </el-form-item>
                    <el-form-item label="Right Source IP" style="min-width:400px;" prop="right_source" class='validate-item'>
                        <el-input v-model.trim='addIPsecConfigForm.right_source' maxlength="64">
                            <template slot="append"><%=rb.getString("FanWei")%>:0~64 Digit string,Support IPv4 or IPv6</template>
                        </el-input>
                    </el-form-item>
                    <el-form-item label="Right Subnet" style="min-width:400px;" prop="right_subnet" class='validate-item'>
                        <el-input v-model.trim='addIPsecConfigForm.right_subnet' maxlength="64">
                            <template slot="append"><%=rb.getString("FanWei")%>:0~64 Digit string</template>
                        </el-input>
                    </el-form-item>
                    <el-form-item v-show="rightSecretKeyShow" label="Right Secret Key" style="min-width:400px;" prop="right_secret_key" class='validate-item'>
                        <el-input v-model.trim='addIPsecConfigForm.right_secret_key' maxlength="64">
                            <template slot="append"><%=rb.getString("FanWei")%>:0~64 Digit string</template>
                        </el-input>
                    </el-form-item>
                </div>
                <div class="bottomLine"></div>
                <div class="addIPsecConfigItemTitleCls">
                    <div class="dotCls"></div>
                    <div><%=rb.getString("GaoJiSheZhi")%></div>
                </div>
                <div class="addIPsecConfigItemBoxCls">
                    <el-form-item label="IKE Encryption" style="min-width:400px;" prop="ike_encryption">
                        <el-select v-model='addIPsecConfigForm.ike_encryption'>
                            <el-option label='aes128' value='aes128'></el-option>
                            <el-option label='aes256' value='aes256'></el-option>
                            <el-option label='3des' value='3des'></el-option>
                            <el-option label='des' value='des'></el-option>
                        </el-select>
                    </el-form-item>
                    <el-form-item label="IKE DH Group" style="min-width:400px;" prop="ike_dh_group">
                        <el-select v-model='addIPsecConfigForm.ike_dh_group'>
                            <el-option label='modp768' value='modp768'></el-option>
                            <el-option label='modp1024' value='modp1024'></el-option>
                            <el-option label='modp1536' value='modp1536'></el-option>
                            <el-option label='modp2048' value='modp2048'></el-option>
                            <el-option label='modp4096' value='modp4096'></el-option>
                        </el-select>
                    </el-form-item>
                    <el-form-item label="IKE Authentication" style="min-width:400px;" prop="ike_authentication">
                        <el-select v-model='addIPsecConfigForm.ike_authentication'>
                            <el-option label='sha1' value='sha1'></el-option>
                            <el-option label='sha1_160' value='sha1_160'></el-option>
                            <el-option label='sha256' value='sha256'></el-option>
                            <el-option label='sha256_96' value='sha256_96'></el-option>
                        </el-select>
                    </el-form-item>
                    <el-form-item label="ESP Encyption" style="min-width:400px;" prop="esp_encryption">
                        <el-select v-model='addIPsecConfigForm.esp_encryption'>
                            <el-option label='aes128' value='aes128'></el-option>
                            <el-option label='aes256' value='aes256'></el-option>
                            <el-option label='3des' value='3des'></el-option>
                            <el-option label='des' value='des'></el-option>
                        </el-select>
                    </el-form-item>
                    <el-form-item label="ESP DH Group" style="min-width:400px;" prop="esp_dh_group">
                        <el-select v-model='addIPsecConfigForm.esp_dh_group'>
                            <el-option label='modp768' value='modp768'></el-option>
                            <el-option label='modp1024' value='modp1024'></el-option>
                            <el-option label='modp1536' value='modp1536'></el-option>
                            <el-option label='modp2048' value='modp2048'></el-option>
                            <el-option label='modp4096' value='modp4096'></el-option>
                        </el-select>
                    </el-form-item>
                    <el-form-item label="ESP Authentication" style="min-width:400px;" prop="esp_authentication">
                        <el-select v-model='addIPsecConfigForm.esp_authentication'>
                            <el-option label='sha1' value='sha1'></el-option>
                            <el-option label='sha1_160' value='sha1_160'></el-option>
                            <el-option label='sha256' value='sha256'></el-option>
                            <el-option label='sha256_96' value='sha256_96'></el-option>
                        </el-select>
                    </el-form-item>
                    <el-form-item label="Key Life" style="min-width:400px;" prop="key_life" class='validate-item'>
                        <el-input v-model.trim='addIPsecConfigForm.key_life'>
                            <template slot="append"><%=rb.getString("ShiJianShuRuTiShi")%></template>
                        </el-input>
                    </el-form-item>
                    <el-form-item label="IKE Life Time" style="min-width:400px;" prop="ike_life_time" class='validate-item'>
                        <el-input v-model.trim='addIPsecConfigForm.ike_life_time'>
                            <template slot="append"><%=rb.getString("ShiJianShuRuTiShi")%></template>
                        </el-input>
                    </el-form-item>
                    <el-form-item label="Rekey Margin" style="min-width:400px;" prop="rekey_margin" class='validate-item'>
                        <el-input v-model.trim='addIPsecConfigForm.rekey_margin'>
                            <template slot="append"><%=rb.getString("ShiJianShuRuTiShi")%></template>
                        </el-input>
                    </el-form-item>
                    <el-form-item label="Dpd Action" style="min-width:400px;" prop="dpd_action">
                        <el-select v-model='addIPsecConfigForm.dpd_action'>
                            <el-option label='none' value='none'></el-option>
                            <el-option label='clear' value='clear'></el-option>
                            <el-option label='hold' value='hold'></el-option>
                            <el-option label='restart' value='restart'></el-option>
                        </el-select>
                    </el-form-item>
                    <el-form-item label="Dpd Delay" style="min-width:400px;" prop="dpd_delay" class='validate-item'>
                        <el-input v-model.trim='addIPsecConfigForm.dpd_delay'>
                            <template slot="append"><%=rb.getString("ShiJianShuRuTiShi")%></template>
                        </el-input>
                    </el-form-item>
                    <el-form-item label="Reauth Time" style="min-width:400px;" prop="reauth_time" class='validate-item'>
                        <el-input maxlength="64" v-model.trim='addIPsecConfigForm.reauth_time'>
                            <template slot="append"><%=rb.getString("FanWei")%>:0~64 Digit string</template>
                        </el-input>
                    </el-form-item>
                    <el-form-item label="Copy Dscp" style="min-width:400px;" prop="copy_dscp">
                        <el-select v-model='addIPsecConfigForm.copy_dscp' clearable>
                            <el-option label='out' value='out'></el-option>
                            <el-option label='yes' value='yes'></el-option>
                            <el-option label='in' value='in'></el-option>
                        </el-select>
                    </el-form-item>
                </div>
            </el-form>
        </div>
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="addIPsecConfigSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="addIPsecConfigShow = false"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
    <!--  IPsec 证书弹窗 -->
	<el-dialog id="addIPsecCert" class="egwSettingAddDialogCls" top="25vh" :title="IPsecCertDialogTitle" width="50%" :visible.sync="certTableBoxShow" :close-on-click-modal="false" append-to-body  @close="certTableBoxClose">		
		<div class="certTableBoxCls">
            <!--:url="CAcertTableUrl"	:data="CAcertTableData" -->
            <el-ctable
                v-if="certTableType == 'CA'"
                id="CAcertTable"
                :url="CAcertTableUrl"
                :query-params="queryCAcertParams" 
                ref="CAcertTable"
                :time="6"
                :height="'100%'"
                :row-key="'file_name'"
                @selection-change='CAcertSelect'
                pagination="true">
                    <!-- 列表toolbar -->
                <template slot="toolbar">
                    <div style="display: flex;align-items: center;position:relative;">
                        <el-query type="normal" @query="queryCAcertList" placeholder="Certificate Name"></el-query>
                        <div class="newIconBoxCls-bt" style="right:20px;top:5px;" @click="verifiedAccountImportClick('CA')" tip="<%=rb.getString("DaoRu")%>">		
							<span class=" el-icon-operation-import el-icon"></span>
						</div>
                        <div :class="clearCaBtnDis == true ? 'newIconBoxCls-bt disabled' : 'newIconBoxCls-bt'" style="right:60px;top:5px;" @click="certDel('','clear','CA')" tip="<%=rb.getString("QingKong")%>">		
							<span class=" el-icon-operation-delete el-icon"></span>
						</div>
                    </div>
                </template>
                    <!-- 列表columns -->
                <el-table-column type="selection" width="45" :reserve-selection="true"></el-table-column>
                <el-table-column prop="file_name" :show-overflow-tooltip="false" label="Certificate Name" min-width="130">
                    <template slot-scope="scope">
                        <el-tooltip popper-class="tooltipCls" :content='scope.row.file_name' placement='bottom'>
                            <div class="beyondEllipsisCls">{{scope.row.file_name}}</div>
                        </el-tooltip>
                    </template>
                </el-table-column>
                <el-table-column prop="file_size" label='<%=rb.getString("WenJianDaXiao")%>' min-width="100">
                    <template slot-scope="scope">
                        <span style="margin-right:10px;">{{scope.row.file_size}}K(Byte)</span>
                    </template>
                </el-table-column>
                <el-table-column prop="file_status" label='<%=rb.getString("ZhuangTai")%>' min-width="100">
                    <template slot-scope="scope">
                        <div v-if="scope.row.file_status == '0'" style="display: flex;align-items: center;">
                            <span class="el-icon el-icon-operation-synchronize UnSyncStatus"></span>
                            <span style="margin-left: 5px;"><%=rb.getString("WeiTongBu")%></span>
                        </div>
                        <div v-if="scope.row.file_status == '1'" style="display: flex;align-items: center;">
                            <span class="el-icon el-icon-operation-synchronize SyncStatus"></span>
                            <span style="margin-left: 5px;"><%=rb.getString("YiTongBu")%></span>
                        </div>
                    </template>
                </el-table-column>
                <el-table-column label='<%=rb.getString("CaoZuo")%>' min-width="120">
                    <template slot-scope="scope">
                        <div style="display: flex;">
                            <el-tooltip popper-class="tooltipCls" content='<%=rb.getString("HuoQu")%>' placement='bottom'>
                                <div v-if="scope.row.file_status == '2'" class="opBtnBoxCls" @click="getCert(scope.row,'CA')"><span class="el-icon el-icon-operation-synchronize greyIcon"></span></div>
                            </el-tooltip>
                            <el-tooltip popper-class="tooltipCls" content='<%=rb.getString("XiaZai")%>' placement='bottom'>
                                <div v-if="scope.row.file_status != '2'" class="opBtnBoxCls" @click="certDownload(scope.row,'CA')"><span class="el-icon el-icon-operation-download greyIcon"></span></div>
                            </el-tooltip>
                            <el-tooltip popper-class="tooltipCls" content='<%=rb.getString("ShanChu")%>' placement='bottom'>
                                <div class="opBtnBoxCls" @click="certDel(scope.row,'del','CA')"><span class="el-icon el-icon-operation-delete greyIcon"></span></div>
                            </el-tooltip>
                        </div>
                    </template>
                </el-table-column>
            </el-ctable>
            <!--:url="IPsecCertTableUrl"	:data="IPsecCertTableData" -->
            <el-ctable
                v-if="certTableType == 'IPsec'"
                id="IPsecCertTable"
                :url="IPsecCertTableUrl"
                :query-params="queryIPsecCertParams" 
                ref="IPsecCertTable"
                :height="'100%'" 
                :time="6"
                :row-key="'file_name'"
                @selection-change='IPsecCertSelect'
                pagination="true">
                    <!-- 列表toolbar -->
                <template slot="toolbar">
                    <div style="display: flex;align-items: center;position:relative;">
                        <el-query type="normal" @query="queryCAcertList" placeholder="Certificate Name"></el-query>
                        <div class="newIconBoxCls-bt" style="right:20px;top:5px;" @click="verifiedAccountImportClick('IPsec')" tip="<%=rb.getString("DaoRu")%>">		
                            <span class=" el-icon-operation-import el-icon"></span>
                        </div>
                        <div :class="clearIPsecBtnDis == true ? 'newIconBoxCls-bt disabled' : 'newIconBoxCls-bt'" style="right:60px;top:5px;" @click="certDel('','clear','IPsec')" tip="<%=rb.getString("QingKong")%>">		
                            <span class=" el-icon-operation-delete el-icon"></span>
                        </div>
                    </div>
                </template>
                    <!-- 列表columns -->
                <el-table-column type="selection" width="45" :reserve-selection="true"></el-table-column>
                <el-table-column prop="" :show-overflow-tooltip="false" label="Certificate Name" min-width="130">
                    <template slot-scope="scope">
                        <el-tooltip popper-class="tooltipCls" :content='scope.row.file_name' placement='bottom'>
                            <div class="beyondEllipsisCls">{{scope.row.file_name}}</div>
                        </el-tooltip>
                    </template>
                </el-table-column>
                <el-table-column prop="file_size" label='<%=rb.getString("WenJianDaXiao")%>' min-width="100">
                    <template slot-scope="scope">
                        <span style="margin-right:10px;">{{scope.row.file_size}}K(Byte)</span>
                    </template>
                </el-table-column>
                <el-table-column prop="file_status" label='<%=rb.getString("ZhuangTai")%>' min-width="100">
                    <template slot-scope="scope">
                        <div v-if="scope.row.file_status == '0'" style="display: flex;align-items: center;">
                            <span class="el-icon el-icon-operation-synchronize UnSyncStatus"></span>
                            <span style="margin-left: 5px;"><%=rb.getString("WeiTongBu")%></span>
                        </div>
                        <div v-if="scope.row.file_status == '1'" style="display: flex;align-items: center;">
                            <span class="el-icon el-icon-operation-synchronize SyncStatus"></span>
                            <span style="margin-left: 5px;"><%=rb.getString("YiTongBu")%></span>
                        </div>
                    </template>
                </el-table-column>
                <el-table-column label='<%=rb.getString("CaoZuo")%>' min-width="120">
                    <template slot-scope="scope">
                        <div style="display: flex;">
                            <el-tooltip popper-class="tooltipCls" content='<%=rb.getString("HuoQu")%>' placement='bottom'>
                                <div v-if="scope.row.file_status == '2'" class="opBtnBoxCls" @click="getCert(scope.row,'IPsec')"><span class="el-icon el-icon-operation-synchronize greyIcon"></span></div>
                            </el-tooltip>
                            <el-tooltip popper-class="tooltipCls" content='<%=rb.getString("XiaZai")%>' placement='bottom'>
                                <div v-if="scope.row.file_status != '2'" class="opBtnBoxCls" @click="certDownload(scope.row,'IPsec')"><span class="el-icon el-icon-operation-download greyIcon"></span></div>
                            </el-tooltip>
                            <el-tooltip popper-class="tooltipCls" content='<%=rb.getString("ShanChu")%>' placement='bottom'>
                                <div class="opBtnBoxCls" @click="certDel(scope.row,'del','IPsec')"><span class="el-icon el-icon-operation-delete greyIcon"></span></div>
                            </el-tooltip>
                        </div>
                    </template>
                </el-table-column>
            </el-ctable>
            <!--:url="IPsecPrivateKeyTableUrl"	:data="IPsecPrivateKeyTableData" -->
            <el-ctable
                v-if="certTableType == 'Key'"
                id="IPsecPrivateKeyTable"
                :url="IPsecPrivateKeyTableUrl"
                :query-params="queryIPsecPrivateKeyParams" 
                ref="IPsecPrivateKeyTable"
                :height="'100%'"
                :time="6"
                :row-key="'file_name'"
                @selection-change='KeyCertSelect'
                pagination="true">
                    <!-- 列表toolbar -->
                <template slot="toolbar">
                    <div style="display: flex;align-items: center;position:relative;">
                        <el-query type="normal" @query="queryCAcertList" placeholder="Certificate Name"></el-query>
                        <div class="newIconBoxCls-bt" style="right:20px;top:5px;" @click="verifiedAccountImportClick('Key')" tip="<%=rb.getString("DaoRu")%>">		
                            <span class=" el-icon-operation-import el-icon"></span>
                        </div>
                        <div :class="clearKeyBtnDis == true ? 'newIconBoxCls-bt disabled' : 'newIconBoxCls-bt'" style="right:60px;top:5px;" @click="certDel('','clear','Key')" tip="<%=rb.getString("QingKong")%>">		
                            <span class=" el-icon-operation-delete el-icon"></span>
                        </div>
                    </div>
                </template>
                    <!-- 列表columns -->
                <el-table-column type="selection" width="45" :reserve-selection="true"></el-table-column>
                <el-table-column prop="file_name" :show-overflow-tooltip="false" label="IPsec Private Key" min-width="130">
                    <template slot-scope="scope">
                        <el-tooltip popper-class="tooltipCls" :content='scope.row.file_name' placement='bottom'>
                            <div class="beyondEllipsisCls">{{scope.row.file_name}}</div>
                        </el-tooltip>
                    </template>
                </el-table-column>
                <el-table-column prop="file_size" label='<%=rb.getString("WenJianDaXiao")%>' min-width="100">
                    <template slot-scope="scope">
                        <span style="margin-right:10px;">{{scope.row.file_size}}K(Byte)</span>
                    </template>
                </el-table-column>
                <el-table-column prop="file_status" label='<%=rb.getString("ZhuangTai")%>' min-width="100">
                    <template slot-scope="scope">
                        <div v-if="scope.row.file_status == '0'" style="display: flex;align-items: center;">
                            <span class="el-icon el-icon-operation-synchronize UnSyncStatus"></span>
                            <span style="margin-left: 5px;"><%=rb.getString("WeiTongBu")%></span>
                        </div>
                        <div v-if="scope.row.file_status == '1'" style="display: flex;align-items: center;">
                            <span class="el-icon el-icon-operation-synchronize SyncStatus"></span>
                            <span style="margin-left: 5px;"><%=rb.getString("YiTongBu")%></span>
                        </div>
                    </template>
                </el-table-column>
                <el-table-column prop=""  label='<%=rb.getString("CaoZuo")%>' min-width="120">
                    <template slot-scope="scope">
                        <div style="display: flex;">
                            <el-tooltip popper-class="tooltipCls" content='<%=rb.getString("HuoQu")%>' placement='bottom'>
                                <div v-if="scope.row.file_status == '2'" class="opBtnBoxCls" @click="getCert(scope.row,'Key')"><span class="el-icon el-icon-operation-synchronize greyIcon"></span></div>
                            </el-tooltip>
                            <el-tooltip popper-class="tooltipCls" content='<%=rb.getString("XiaZai")%>' placement='bottom'>
                                <div v-if="scope.row.file_status != '2'" class="opBtnBoxCls" @click="certDownload(scope.row,'Key')"><span class="el-icon el-icon-operation-download greyIcon"></span></div>
                            </el-tooltip>
                            <el-tooltip popper-class="tooltipCls" content='<%=rb.getString("ShanChu")%>' placement='bottom'>
                                <div class="opBtnBoxCls" @click="certDel(scope.row,'del','Key')"><span class="el-icon el-icon-operation-delete greyIcon"></span></div>
                            </el-tooltip>
                        </div>
                    </template>
                </el-table-column>
            </el-ctable>
        </div>
	</el-dialog>
    <!-- 证书导入 弹窗 -->
	<el-dialog id="CertImport" class="egwSettingAddDialogCls" title="<%=rb.getString("DaoRu")%>" top="30vh" width="480px" :visible.sync="verifiedAccountImportShow" class="importCard" :close-on-click-modal="false" append-to-body @close="closeImportParams">		
		<el-form label-position="top" ref="verifiedAccountImportForm" :model='verifiedAccountImportForm' :rules='verifiedAccountImportRules'>     		     			            
        	<el-form-item label="<%=rb.getString("DaoRuWenJian")%>" label-width="110px" prop="" style="margin-bottom:0px;">
               	<el-upload 
             		ref="verifiedAccountUpload"
             		:before-upload='verifiedAccountBeforeUpload' 
             		:on-success='verifiedAccountCheckFile' 
             		:on-change="verifiedAccountFileChange"  
             		:show-file-list=false 	                  		
				    :action="verifiedAccountImportForm.uploadFileUrl" 
				    :data="verifiedAccountImportFileParams" 
				    name="uploadFile" 
				    :accept="acceptType"
				    :auto-upload="false">
					<el-input :value="verifiedAccountImportFileName" placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>' style="width:320px;">
						<a slot="suffix" class="el-icon el-icon-operation-import importBox" @click="verifiedAccountImportFileSelect"></a>
					</el-input>
					<div slot="tip" :class="verifiedAccountImportForm.errMassageShow? 'el-upload__tip':'fileAcceptTip'">{{verifiedAccountImportForm.errMassage}}</div>
					<a slot="trigger" ref="file_up"></a>
				</el-upload>	
         	</el-form-item>
            <el-form-item style="width:100%;" v-if="importType == 'user' || importType == 'block'">
         		<div style='color:#999999;'>
					<span style="cursor:pointer;" @click="verifiedAccountExportTemplate">
						<span class='el-icon el-icon-common-download'></span>
						<span style='color:#363B4E;text-decoration:underline'><%=rb.getString("DaoChuMuBan")%></span>
					</span>
				</div> 
         	</el-form-item>   
        </el-form> 
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="verifiedAccountUploadParams"><%=rb.getString("QueDing")%></el-button>
              <el-button @click="closeImportParams"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
    <!-- 认证用户新增 弹窗 -->
	<el-dialog id="VerifiedAccount" class="egwSettingAddDialogCls" top="25vh" title="<%=rb.getString("TianJia")%>" width="50%" :visible.sync="addVerifiedAccountBoxShow" @close="addVerifiedAccountClose" :close-on-click-modal="false" append-to-body>		
		<el-form label-position="top" ref="addVerifiedAccountForm" :model='addVerifiedAccountForm' :rules='addVerifiedAccountRules'>
            <el-form-item label="IMSI" label-width="110px" prop="imsi">
                <span slot="label" class="labelSlotCls">
                    IMSI
                    <span><%=rb.getString("15Wei10JinZhiShu")%></span>
                </span>
                <el-input :disabled="optVerifiedAccountType == 'edit'" v-model.trim='addVerifiedAccountForm.imsi' style="width:316px;"></el-input>
            </el-form-item>
            <el-form-item label="KEY" label-width="110px" prop="key">
                <span slot="label" class="labelSlotCls">
                    KEY
                    <span><%=rb.getString("32Wei16JinZhiShu")%></span>
                </span>
                <el-input v-model.trim='addVerifiedAccountForm.key' style="width:316px;"></el-input>
            </el-form-item>
            <el-form-item label="OPC" label-width="110px" prop="opc">
                <span slot="label" class="labelSlotCls">
                    OPC
                    <span><%=rb.getString("32Wei16JinZhiShu")%></span>
                </span>
                <el-input v-model.trim='addVerifiedAccountForm.opc' style="width:316px;"></el-input>
            </el-form-item>
            <div style="width: 100%;">
                <div style="margin-bottom: 10px;">
                    <el-checkbox v-model="addVerifiedAccountForm.showAdditionalCol" true-label="1" false-label="0"></el-checkbox>
                    <%=rb.getString("TongShiXiaFaYiXiaCanShu")%>
                </div>
                <div style="width: 100%;display:flex;flex-wrap:wrap;border:1px solid #DFE2EE;padding:10px;box-sizing:border-box;">
                    <el-form-item label="Sn" label-width="110px" prop="userSn">
                        <span slot="label" class="labelSlotCls">
                            Sn
                            <span>Length:0~250,String</span>
                        </span>
                        <el-input v-model.trim='addVerifiedAccountForm.userSn' maxlength="250" style="width:316px;"></el-input>
                    </el-form-item>
                    <el-form-item label="Mac" label-width="110px" prop="userMac">
                        <span slot="label" class="labelSlotCls">
                            Mac
                            <span>Example: 1A:2B:3C:4D:5E:6F</span>
                        </span>
                        <el-input v-model.trim='addVerifiedAccountForm.userMac' maxlength="250" style="width:316px;"></el-input>
                    </el-form-item>
                    <el-form-item label="Rc Const Id" label-width="110px" prop="rcConstId">
                        <span slot="label" class="labelSlotCls">
                            Rc Const Id
                            <span>Length:1~10,<%=rb.getString("ZhengXing")%></span>
                        </span>
                        <el-input v-model.trim='addVerifiedAccountForm.rcConstId' maxlength="250" style="width:316px;"></el-input>
                    </el-form-item>
                    <el-form-item label="Vendor" label-width="110px" prop="vendor">
                        <span slot="label" class="labelSlotCls">
                            Vendor
                            <span>Length:0~64,String</span>
                        </span>
                        <el-input v-model.trim='addVerifiedAccountForm.vendor' maxlength="64" style="width:316px;"></el-input>
                    </el-form-item>
                </div>
            </div>
                

            
        </el-form>
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="addVerifiedAccountSubmit"><%=rb.getString("QueDing")%></el-button>
			<el-button @click="addVerifiedAccountClose"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
     <!--  BlockList 弹窗 -->
	<el-dialog id="BlockList"  class="egwSettingAddDialogCls" top="25vh" title="<%=rb.getString("HeiMingDan")%>" width="50%" :visible.sync="blockListBoxShow" @close="addBlockListClose" :close-on-click-modal="false" append-to-body>		
		<div class="certTableBoxCls">
           <el-ctable
                id="blockListTable"
                :url="blockListTableUrl"
                :query-params="queryBlockListParams" 
                ref="blockListTable" 
                :time="6"
                :height="'100%'" 
                pagination="true">
                    <!-- 列表toolbar -->
                <template slot="toolbar">
                    <div class="toolbarBoxCls" style="position:relative;display:flex;">
                        <el-query type="normal" @query="queryBlockList" placeholder="<%=rb.getString("IMSI")%>"></el-query>
                        <span class="el-icon el-icon-operation-delete" style="position:relative;top:9px;margin-left:20px;" @click="blockListDel('','clear')"></span>
                        <span class="el-icon el-icon-circle-add" style="position:absolute;right:60px;top:11px;" @click="addBlockListClick"></span>
                        <span class="el-icon el-icon-operation-import" style="position:absolute;right:30px;top:9px;" @click="verifiedAccountImportClick('block')"></span>
                        <span class="el-icon el-icon-operation-export" style="position:absolute;right:0px;top:9px;" @click="exportBlockList"></span>
                    </div>
                </template>
                    <!-- 列表columns -->
                <el-table-column prop="imsi" show-overflow-tooltip label="IMSI" min-width="130">
                    <template slot-scope="scope">
                        <div style="display: flex;align-items: center;justify-content: space-between;">
                            <span>{{scope.row.imsi}}</span>
                            <el-tooltip popper-class="tooltipCls" content='<%=rb.getString("ShanChu")%>' placement='bottom'>
                                <div class="opBtnBoxCls" @click="blockListDel(scope.row,'del')"><span class="el-icon el-icon-operation-delete greyIcon"></span></div>
                            </el-tooltip>
                        </div>
                    </template>
                </el-table-column>
            </el-ctable>
        </div>
	</el-dialog>
    <!-- USIM黑名单新增 弹窗 -->
	<el-dialog class="addBlockListDialog" title="<%=rb.getString("TianJia")%>" top="30vh" width="480px" :visible="addBlockListDialogShow" :close-on-click-modal="false" append-to-body @close="closeAddBlockList">		
		<el-form label-position="top" ref="addBlockListForm" :model='addBlockListForm' :rules='addBlockListRules'>     		     			            
        	<el-form-item label="IMSI" label-width="110px" prop="imsi" style="margin-left:20px;">
				<span slot="label" class="labelSlotCls">
					IMSI
					<span><%=rb.getString("15Wei10JinZhiShu")%></span>
				</span>
				<el-input v-model.trim='addBlockListForm.imsi' style="width:316px;"></el-input>
			</el-form-item>
         	<el-form-item label="" v-if="false">
         		<el-input style="width:316px;"></el-input>
         	</el-form-item>      	    
        </el-form> 
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="addBlockListSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="closeAddBlockList"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
</div>

<script>
var regIp = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/,
    regKey = /^[A-Fa-f0-9]{32}$/,
    regNumber = /^[0-9]{15}$/;
var egwSecurityGatewayPage = new Vue({
	el: '#egwSecurityGatewayPage', 
	data() {
		var vm = this;
        var validate_TimeStr = (rule,value,callback)=>{
                var reg = /^\+?[1-9][0-9]*$/;
                if(value == '' || value == undefined || value == null){
                    callback()
                }else{
                    if(value.length < 2){
                        callback(new Error('<%=rb.getString("ShiJianShuRuTiShi")%>'))
                    }else{
                        var lastStr = value.charAt(value.length - 1);
                        var delLastStr = value.slice(0,value.length - 1);
                        if((lastStr == 'd' || lastStr == 'h' || lastStr == 'm' || lastStr == 's') && reg.test(delLastStr)){
                            callback();
                        }else{
                            callback(new Error('<%=rb.getString("ShiJianShuRuTiShi")%>'))
                        }
                    }
                }
            },
            validateIPorMask = function(rule,value,callback) {
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
			},
            validateIMSI = function(rule,value,callback) {
		        	
                if( value === '' || value === null || value === undefined) {
                    callback('<%=rb.getString("15Wei10JinZhiShu")%>');
                }else {
                    if(regNumber.test(value)) {
                        callback();
                    }else{
                        callback('<%=rb.getString("15Wei10JinZhiShu")%>');
                    }
                }
            },
            validateKEY = function(rule,value,callback) {
                
                if( value === '' || value === null || value === undefined) {
                    callback('<%=rb.getString("32Wei16JinZhiShu")%>');
                }else {
                    if(regKey.test(value)) {
                        callback();
                    }else{
                        callback('<%=rb.getString("32Wei16JinZhiShu")%>');
                    }
                }
            },
            validateOPC = function(rule,value,callback) {
                
                if( value === '' || value === null || value === undefined) {
                    callback('<%=rb.getString("32Wei16JinZhiShu")%>');
                }else {
                    if(regKey.test(value)) {
                        callback();
                    }else{
                        callback('<%=rb.getString("32Wei16JinZhiShu")%>');
                    }
                }
            },
            validateUserMac = function(rule,value,callback) {
                var regMac = /^[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}$/;
                if( value === '' || value === null || value === undefined) {
                    callback();
                }else {
                    if(regMac.test(value)) {
                        callback();
                    }else{
                        callback('Example: 1A:2B:3C:4D:5E:6F');
                    }
                }
            },
            validateRcConstId = function(rule,value,callback) {
                var regRcConstId = /^([1-9]{1}[0-9]{0,9})$/;
                if( value === '' || value === null || value === undefined) {
                    callback();
                }else {
                    if(regRcConstId.test(value)) {
                        callback();
                    }else{
                        callback('Length:10,<%=rb.getString("ZhengXing")%>');
                    }
                }
            };
		return {
            egwCode:'',
            egwSn:'',
            activeName:'IPsec',
            queryIPsecConfigParams:{
                searchText:'',
            },
            IPsecConfigTableUrl:'',
            addIPsecConfigShow:false,
            addIPsecConfigCertData:[],
            addIPsecConfigForm:{
                group_name:'',
                switch_enable:'1',
                secret_key:'',
                fragmentation:'yes',

                left_ip:'',
                left_id:'',
                left_auth:'psk',
                left_source:'',
                left_subnet:'',
                left_cert:'',

                right_ip:'',
                right_id:'',
                right_auth:'psk',
                right_source:'',
                right_subnet:'',
                right_secret_key:'',

                ike_encryption:'aes128',
                ike_dh_group:'modp768',
                ike_authentication:'sha1',
                esp_encryption:'aes128',
                esp_dh_group:'modp768',
                esp_authentication:'sha1',
                key_life:'',
                ike_life_time:'',
                rekey_margin:'',
                dpd_action:'none',
                dpd_delay:'',
                reauth_time:'',
                copy_dscp:'',
            },	
            addIPsecConfigRules:{
                group_name:[{required:true,message:'String,Length：1~64',trigger:'blur'}],
                key_life:[{validator: validate_TimeStr}],
                ike_life_time:[{validator: validate_TimeStr}],
                rekey_margin:[{validator: validate_TimeStr}],
                dpd_delay:[{validator: validate_TimeStr}]
            },
            addIPsecConfigType:'',
            IPsecConfigRowData:'',
            IPsecCertForm:{
                cert_manage:'Enable',
                cmp_server_address:'',
                country:'',
                ca_cert:'',
                ipsec_cert:'',
                private_key:'',
                file_name:'',
                valid_start_time:'',
                valid_end_time:'',
                refresh_time:'',
                refresh_status:''
            },
            IPsecCertRules:{
                cmp_server_address:[{min:0,max:512,trigger:'change'}]
            },
            CAcertSelection:[],
            IPsecCertSelection:[],
            KeyCertSelection:[],
            ca_certList:[],
            ipsec_certList:[],
            private_keyList:[],
            certTableBoxShow:false,
			certTableType:'',
            CAcertTableUrl:'${ctx}/egw/ipsec/config/getCaCertFileList.action',
            queryCAcertParams:{
                searchText:'',
                serial_number:''
            },
            IPsecCertTableUrl:'${ctx}/egw/ipsec/config/getCertFileList.action',
            queryIPsecCertParams:{
                searchText:'',
                serial_number:''
            },
            IPsecPrivateKeyTableUrl:'${ctx}/egw/ipsec/config/getPrivateKeyFileList.action',
            queryIPsecPrivateKeyParams:{
                searchText:'',
                serial_number:''
            },
            tableCode:{
                'CA':'CAcertTable',
                'IPsec':'IPsecCertTable',
                'Key':'IPsecPrivateKeyTable'
            },
            verifiedAccountTableUrl:'',
            queryVerifiedAccountParams:{
                searchText:'',
            },
            verifiedAccountSelection:[],
            bulkTableMessage:{title:'<%=rb.getString("YiXuan")%>',subTitle:'IMSI',clear:'<%=rb.getString("QingKong")%>',cancel:'<%=rb.getString("QuXiao")%>'},
            verifiedAccountRowData:{},
            verifiedAccountMenus:[],
            optVerifiedAccountType:'',
            importType:'',
            acceptType:'.csv',
            verifiedAccountImportShow:false,
            verifiedAccountImportForm: {
                uploadFileUrl: '',
                errMassageShow:false,
                errMassage:'Only .csv is supported'
            },	   
            verifiedAccountImportRules: {},
            verifiedAccountImportFileParams:{},              
            verifiedAccountImportFileName:'',
            addVerifiedAccountBoxShow:false,
            addVerifiedAccountForm:{
                imsi:'',
                key:'',
                opc:'',
                userSn:'',
                userMac:'',
                rcConstId:'',
                vendor:'',
                showAdditionalCol:'0'
            },
            addVerifiedAccountRules:{
                imsi:[{validator: validateIMSI}],
                key:[{validator: validateKEY}],
                opc:[{validator: validateOPC}],
                userMac:[{validator: validateUserMac}],
                rcConstId:[{validator: validateRcConstId}],
            },
            blockListBoxShow:false,
            blockListTableUrl:'',
            queryBlockListParams:{
                searchText:''
            },
            addBlockListDialogShow:false,
            addBlockListForm:{
                imsi:''
            },
            addBlockListRules:{
                imsi:[
                    {validator: validateIMSI},
                ],
            },
		};
	},
	computed: {
        // 获取证书信息 label class
        fetchLabelClass(){
            return language == 'en'? 'fetchLabelEnCls' : 'fetchLabelZhCls';
        },
        IPsecCertDialogTitle(){
            var codes ={
                'CA':'<%=rb.getString("CAZhengShu")%>',
                'IPsec':'<%=rb.getString("eGWIPsecZhengShu")%>',
                'Key':'IPsec Private Key'
            }
            return codes[this.certTableType];
        },
        clearCaBtnDis(){
            var vm = this;
            var fileList = vm.CAcertSelection;
            return fileList.length == 0 ? true : false ;
        },
        clearIPsecBtnDis(){
            var vm = this;
            var fileList = vm.IPsecCertSelection;
            return fileList.length == 0 ? true : false ;
        },
        clearKeyBtnDis(){
            var vm = this;
            var fileList = vm.KeyCertSelection;
            return fileList.length == 0 ? true : false ;
        },
        issueAndUpdateCertBtnDis(){
            return this.IPsecCertForm.ca_cert == '' || this.IPsecCertForm.ipsec_cert == '' || this.IPsecCertForm.private_key == ''
        },
        secretKeyShow(){
            return (this.addIPsecConfigForm.left_auth == 'psk' && this.addIPsecConfigForm.right_auth == 'psk') || this.addIPsecConfigForm.left_auth == 'pubkey' || this.addIPsecConfigForm.right_auth == 'pubkey'
        },
        leftCertShow(){
            return this.addIPsecConfigForm.left_auth == 'pubkey' || this.addIPsecConfigForm.right_auth == 'pubkey'
        },
        rightSecretKeyShow(){
            return this.addIPsecConfigForm.left_auth != 'psk' && this.addIPsecConfigForm.right_auth == 'psk'
        },
    },
	methods: {
        // 初始化
		init(row,code,sn,status){
		    var vm =this;
			vm.egwCode = code;
            vm.egwSn = sn;
            vm.IPsecConfigTableUrl = '${ctx}/egw/ipsec/config/getConfigList.action?egw_code='+ vm.egwCode;
            vm.blockListTableUrl = '${ctx}/egw/ipsec/getUsimBlackList.action?egw_code='+vm.egwCode;
            vm.verifiedAccountTableUrl = '${ctx}/egw/ipsec/getUserList.action?egw_code='+ vm.egwCode;
            vm.queryCAcertParams.serial_number = vm.egwSn;
            vm.queryIPsecCertParams.serial_number = vm.egwSn;
            vm.queryIPsecPrivateKeyParams.serial_number = vm.egwSn;
		},
        // Tab 切换
		tabClick(){
            var vm = this,
                str = Math.random().toString();
            if(vm.activeName == 'IPsec'){
                vm.IPsecConfigTableUrl = '${ctx}/egw/ipsec/config/getConfigList.action?egw_code='+ vm.egwCode + "&randomCode="+ str;
            }else if(vm.activeName == 'Cert'){
                vm.getCertInfo();
                vm.getFileListData();
                try{
                    clearInterval(certInfoTimer);
                }catch(e){}
                certInfoTimer = setInterval(function() {
                    var tb = $("#IPsecCertBox");
                    
                    if(tb.length) {
                        vm.getCertInfo();
                    }else {
                        clearInterval(certInfoTimer);
                    }
                },6000);
            }else{
                vm.verifiedAccountTableUrl = '${ctx}/egw/ipsec/getUserList.action?egw_code='+ vm.egwCode + "&randomCode="+ str;
            }
            if(vm.activeName != 'Cert'){
                try{
                    clearInterval(certInfoTimer);
                }catch(e){}
            }
        },
        // IPsec配置表格 模糊查询
        queryIPsecConfig(val){
            var vm = this;
            vm.queryIPsecConfigParams.searchText= val;
        },
        // IPsec配置 新增
        addIPsecConfigClick(){
            var vm = this;
            vm.addIPsecConfigType = 'add';
            axios.post("${ctx}/egw/ipsec/config/getAllCertFile.action",stringify({
                serial_number:vm.egwSn,
                file_type:'ipseccert'
            })).then(function(response){
                var data = response.data;
                if(data && data.length > 0){
                    var leftCertData = [];
                    data.map((item)=>{
                        if(item.file_status != '0'){
                            leftCertData.push(item)
                        }
                    })
                    vm.addIPsecConfigCertData = leftCertData;
                }
            });
            vm.addIPsecConfigShow = true;
            vm.$nextTick(()=>{
                initForm(vm.$refs.addIPsecConfigForm)
            })
        },
        // IPsec配置 修改
        IPsecConfigModify(row){
            var vm = this,
                params={
                    egw_code:vm.egwCode,
                    index:row.index
                };
            vm.IPsecConfigRowData = row;
            vm.addIPsecConfigType = 'edit';
            axios.post("${ctx}/egw/ipsec/config/getAllCertFile.action",stringify({
                serial_number:vm.egwSn,
                file_type:'ipseccert'
            })).then(function(response){
                var data = response.data;
                if(data && data.length > 0){
                    var leftCertData = [];
                    data.map((item)=>{
                        if(item.file_status != '0'){
                            leftCertData.push(item)
                        }
                    })
                    vm.addIPsecConfigCertData = leftCertData;
                }
                axios.post('${ctx}/egw/ipsec/config/getConfigInfo.action',stringify(params)).then(function(response){
                var data = response.data;
                    if(data){
                        Object.assign(vm.addIPsecConfigForm,data);
                        var isExist = false;
                        vm.addIPsecConfigCertData.map((item)=>{
                            if(item.file_name == vm.addIPsecConfigForm.left_cert){
                                isExist = true;
                            }
                        })
                        if(!isExist){
                            vm.addIPsecConfigForm.left_cert = '';
                        }
                        
                        vm.addIPsecConfigShow = true;
                        vm.$nextTick(()=>{
                            initForm(vm.$refs.addIPsecConfigForm)
                        })
                    } 
                }) 
            });
        },
        // IPsec配置 删除
        IPsecConfigDel(row){
            var vm = this,
                urls = '${ctx}/egw/ipsec/config/delConfig.action',
                params={
                    egw_code:vm.egwCode,
                    index:row.index
                };
            vm.$confirm('<%=rb.getString("QueRenShanChu")%>','<%=rb.getString("QueRen")%>',{
                confirmButtonText:'<%=rb.getString("QueDing")%>',
                cancalButtonText:'<%=rb.getString("QuXiao")%>',
                type:'warning'
            }).then(()=>{
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
            }).catch(()=>{})
        },
        // IPsec配置 开关事件
        IPsecConfigEnableChange(row){
            var vm = this,
                urls = '${ctx}/egw/ipsec/config/switchEnable.action'
                params={
                    egw_code:vm.egwCode,
                    index:row.index,
                    switch_enable:row.switch_enable == '1' ? '0' : '1'
                };
            axios.post(urls,stringify(params)).then(res=>{
                var data = res.data;
                if(data["success"]){
                    vm.$message({
                        message: '<%=rb.getString("MingLingYiXiaFa")%>',
                        type:'success',
                    });
                    vm.$refs.IPsecConfigTable.refresh();
                }else{
                    vm.$message.error(data["message"])
                }
            })
        },
        // 新增 IPsec配置 提交
        addIPsecConfigSubmit(){
            var vm = this,
                urls = ''
                params={
                    egw_code:vm.egwCode
                };
            if(vm.addIPsecConfigType == 'edit'){
                params.index = vm.IPsecConfigRowData.index
                urls = '${ctx}/egw/ipsec/config/updateConfig.action';
            }else{
                urls = '${ctx}/egw/ipsec/config/addConfig.action';
            }
            Object.assign(params,vm.addIPsecConfigForm);
            if(!vm.secretKeyShow){
                params.secret_key = '';
            }
            if(!vm.leftCertShow){
                params.left_cert = '';
            }
            if(!vm.rightSecretKeyShow){
                params.right_secret_key = '';
            }
            vm.$refs.addIPsecConfigForm.validate(function(valid){
                if(valid){
                    axios.post(urls,stringify(params)).then(res=>{
                        var data = res.data;
                        if(data["success"]){
                            vm.$message({
                                message: '<%=rb.getString("MingLingYiXiaFa")%>',
                                type:'success',
                            });
                            vm.addIPsecConfigShow = false;
                        }else{
                            vm.$message.error(data["message"])
                        }
                    })
                }else{
                    return false
                }
            })
        },
        // 新增 IPsec配置 取消
        closeAddIPsecConfig(){
            var vm = this,
                params = {
                    group_name:'',
                    switch_enable:'1',
                    secret_key:'',
                    fragmentation:'yes',

                    left_ip:'',
                    left_id:'',
                    left_auth:'psk',
                    left_source:'',
                    left_subnet:'',
                    left_cert:'',

                    right_ip:'',
                    right_id:'',
                    right_auth:'psk',
                    right_source:'',
                    right_subnet:'',
                    right_secret_key:'',

                    ike_encryption:'aes128',
                    ike_dh_group:'modp768',
                    ike_authentication:'sha1',
                    esp_encryption:'aes128',
                    esp_dh_group:'modp768',
                    esp_authentication:'sha1',
                    key_life:'',
                    ike_life_time:'',
                    rekey_margin:'',
                    dpd_action:'none',
                    dpd_delay:'',
                    reauth_time:'',
                    copy_dscp:'',
                };
            
            Object.assign(vm.addIPsecConfigForm,params)
            vm.$refs.addIPsecConfigForm.clearValidate();
        },
        // 同步WCG获取证书信息
        refreshCertInfo(){
            var vm = this,
                urls = '${ctx}/egw/ipsec/config/refreshCertInfo.action',
                params = {
                    egw_code:vm.egwCode,
                    serial_number:vm.egwSn,
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
        // 保存证书
        saveCertInfo(){
            var vm = this,
                urls = '${ctx}/egw/ipsec/config/saveCertInfo.action',
                params = {
                    egw_code:vm.egwCode,
                    serial_number:vm.egwSn,
                    cert_manage: vm.IPsecCertForm.cert_manage,
                    cmp_server_address:vm.IPsecCertForm.cmp_server_address,
                    country:vm.IPsecCertForm.country,
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
        // IPsec证书下拉改变
        ipsecCertChange(val){
            var vm = this;
            vm.ipsec_certList.map((item)=>{
                if(item.file_name == val){
                    vm.IPsecCertForm.file_name = item.file_name;
                    vm.IPsecCertForm.valid_start_time = item.valid_start_time;
                    vm.IPsecCertForm.valid_end_time = item.valid_end_time;
                }
            })
        },
        // 查看证书表格
        viewCertClick(type){
            var vm = this;
            if(vm.certTableType == type)return
            vm.certTableBoxShow = true;
            if(vm.certTableType){
                vm.$refs[vm.tableCode[vm.certTableType]].clearSelection();
            }
            vm.certTableType = type;
        },
        // 左侧文件列表关闭
        certTableBoxClose(){
            var vm = this;

            if(vm.certTableType){
                vm.$refs[vm.tableCode[vm.certTableType]].clearSelection();
            }
            vm.certTableBoxShow = false
            vm.certTableType = '';
        },
        // 刷新证书信息
        getCertInfo(){
            var vm = this,
                urls = '${ctx}/egw/ipsec/config/getCertInfo.action',
                params = {
                    egw_code:vm.egwCode,
                    timeZone:timeZone
                };
            axios.post(urls,stringify(params)).then(res=>{
                var data = res.data;
                ['cert_manage','cmp_server_address','country','refresh_time','refresh_status'].map((item)=>{
                    if(data[item]){
                        vm.IPsecCertForm[item] = data[item];
                    }
                })
            })
        },
        // 获取IPsec 文件下拉数据
        getFileListData(){
            var vm = this;

            axios.post("${ctx}/egw/ipsec/config/getAllCertFile.action",stringify({
                serial_number:vm.egwSn,
                timeZone:timeZone,
                file_type:'cacert'
            })).then(function(response){
                var data = response.data;
                if(data && data.length > 0){
                    vm.ca_certList = data;
                }
            });
            axios.post("${ctx}/egw/ipsec/config/getAllCertFile.action",stringify({
                serial_number:vm.egwSn,
                timeZone:timeZone,
                file_type:'ipseccert'
            })).then(function(response){
                var data = response.data;
                if(data && data.length > 0){
                    vm.ipsec_certList = data;
                }
            });
            axios.post("${ctx}/egw/ipsec/config/getAllCertFile.action",stringify({
                serial_number:vm.egwSn,
                timeZone:timeZone,
                file_type:'privatekey'
            })).then(function(response){
                var data = response.data;
                if(data && data.length > 0){
                    vm.private_keyList = data;
                }
            })
        },
        // CA证书表格选择
        CAcertSelect(selection){
            var vm = this;
            vm.CAcertSelection = selection;
        },
        // IPsec证书表格选择
        IPsecCertSelect(selection){
            var vm = this;
            vm.IPsecCertSelection = selection;
        },
        //  key 文件表格选择
        KeyCertSelect(selection){
            var vm = this;
            vm.KeyCertSelection = selection;
        },
        // CA证书表格 模糊搜索
        queryCAcertList(val){
            var vm = this;
            vm.queryCAcertParams.searchText = val;
        },
        // IPsec证书表格 模糊搜索
        queryIPsecCertList(val){
            var vm = this;
            vm.queryIPsecCertParams.searchText = val;
        },
        // IPsec私钥表格 模糊搜索
        queryIPsecPrivateKeyList(){
            var vm = this;
            vm.queryIPsecPrivateKeyParams.searchText = val;
        },
        // 从WCG获取证书
        getCert(row,type){
            var vm = this,
                urls = '${ctx}/egw/ipsec/config/getFile.action'
                params={
                    egw_code:vm.egwCode,
                    serial_number:vm.egwSn,
                };
            if(type == 'CA'){
                params.ca_cert = row.file_name;
            }else if(type == 'IPsec'){
                params.ipsec_cert = row.file_name;
            }else if(type == 'Key'){
                params.private_key = row.file_name;
            }
            axios.post(urls,stringify(params)).then(res=>{
                var data = res.data;
                if(data["success"]){
                    vm.$message({
                        message: '<%=rb.getString("MingLingYiXiaFa")%>',
                        type:'success',
                    });
                    vm.$refs[vm.tableCode[type]].refresh();
                }else{
                    vm.$message.error(data["message"])
                }
            })
        },
        // 证书删除 delType: del-单个删除 clear-删除当前页   fileType :  CA  IPsec Key
        certDel(row,delType,fileType){
            var vm = this,
                urls = '${ctx}/egw/ipsec/config/delFile.action',
                code = {
                    'CA':'cacert',
                    'IPsec':'ipseccert',
                    'Key':'privatekey'
                },
                params={
                    egw_code:vm.egwCode,
                    serial_number:vm.egwSn,
                    file_name:'',
                    file_type:code[fileType]
                };
            if(delType == 'del'){
                if(row.file_status == '0'){
                    var confirmHint = '<%=rb.getString("QueDingShanChuWenJian")%>';
                }else{
                    var confirmHint ='<div style="font-size:14px;color:#333333">'+ '<%=rb.getString("QueDingShanChuWenJian")%>' +'</div>'+'<div style="font-size:12px;color:#999999">'+ '<%=rb.getString("WCG_ShanChuWenJianTiShi")%>' +'</div>';
                }
                params.file_name = row.file_name
            }else{
                
                var fileNameList = [];
                var fileList = vm.$refs[vm.tableCode[fileType]].getChecked();
                if(fileList.length == 0)return
                var confirmHint ='<div style="font-size:14px;color:#333333">'+ '<%=rb.getString("QueDingShanChuWenJian")%>' +'</div>'+'<div style="font-size:12px;color:#999999">'+ '<%=rb.getString("WCG_PiLiangShanChuWenJianTiShi")%>' +'</div>';
                params.file_name = fileList.join(';');
            }
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
                        if(delType == 'del'){
                            vm.$refs[vm.tableCode[fileType]].toggleRowSelection(row,false);
                        }else{
                            vm.$refs[vm.tableCode[fileType]].clearSelection();
                        }
                    }else{
                        vm.$message.error(data["message"])
                    }
                })
            }).catch(()=>{})

        },
        // 证书下载
        certDownload(row,type){
            var vm = this,
                urls = '',
                params = {
                    serial_number:vm.egwSn,
                    egw_code:vm.egwCode,
                    file_name:row.file_name
                };
            if(type == 'CA'){
                urls = '${ctx}/egw/ipsec/config/downloadCaCertFile.action';
            }else if(type == 'IPsec'){
                urls = '${ctx}/egw/ipsec/config/downloadCertFile.action';
            }else if(type == 'Key'){
                urls = '${ctx}/egw/ipsec/config/downloadPrivateKeyFile.action';
            }
            exportByForm(urls,params);
        },
        verifiedAccountImportClick(type){
            var vm =this,
                code={
                    'user' : '.csv',
					'block' : '.csv',
                    'CA' : '.crt',
                    'IPsec' : '.crt',
                    'Key' : '.key'
                };
            vm.importType = type;
            vm.acceptType = code[type];
            if(vm.importType == 'user' || vm.importType == 'block'){
                vm.verifiedAccountImportForm.errMassage = 'Only .csv is supported'
            }else if(vm.importType == 'CA' || vm.importType == 'IPsec'){
                vm.verifiedAccountImportForm.errMassage = 'Only .crt is supported'
            }else if(vm.importType == 'Key'){
                vm.verifiedAccountImportForm.errMassage = 'Only .key is supported'
            }
            vm.verifiedAccountImportShow = true;
        },
        // 选择文件
        verifiedAccountImportFileSelect(){  
            var vm =this;
            vm.$refs.verifiedAccountUpload.clearFiles();
            vm.$refs['file_up'].click();
        },
        /**
        * 文件上传之前
        * @param file{object}   文件信息
        */ 
        verifiedAccountBeforeUpload(file){
            var vm = this, 
                urls = '', 
                fileName = file.name,
                fileSize = file.size,
                fd = new FormData(),
                config = {
                    headers: { 'Content-Type': 'multipart/form-data' }
                },
                code={
                    'user' : 'verifiedAccountTable',
					'block' : 'blockListTable',
                    'CA':'CAcertTable',
                    'IPsec':'IPsecCertTable',
                    'Key':'IPsecPrivateKeyTable'
                };
            if(vm.importType == 'user'){
                urls = '${ctx}/egw/ipsec/uploadUserFile.action';
            }else if(vm.importType == 'block'){
                urls = '${ctx}/egw/ipsec/uploadBlackListFile.action';
            }else if(vm.importType == 'CA'){
                urls = '${ctx}/egw/ipsec/config/uploadCaCertFile.action';
            }else if(vm.importType == 'IPsec'){
                urls = '${ctx}/egw/ipsec/config/uploadCertFile.action';
            }else if(vm.importType == 'Key'){
                urls = '${ctx}/egw/ipsec/config/uploadPrivateKeyFile.action';
            }
            fd.append('egwCode',vm.egwCode); 
            fd.append('uploadFile',file); //文件流
            if(vm.importType != 'user' && vm.importType != 'block'){
                fd.append('serial_number',vm.egwSn); 
                fd.append('file_size',fileSize); 
            }
            axios.post(urls,fd,config).then(function(res){
                var data = res.data;
                if(data["success"]){	
                    if(vm.importType != 'user' && vm.importType != 'block'){
                        vm.getFileListData();
                        vm.$message.success('<%=rb.getString("ChengGong")%>');
                    }else{
                        vm.$message.success('<%=rb.getString("MingLingYiXiaFa")%>');
                    }
                    vm.$refs[code[vm.importType]].refresh();
                    vm.closeImportParams();						
                }else{
                    vm.$message.error(data["message"]);
                    vm.closeFileSelect();
                } 
            })			
            return false;
        },
        //导入
        verifiedAccountCheckFile(res,file){  
            var vm = this;
            if(res.success){
                if(res.suc_count>0){
                    vm.$message({
                        type: 'success',
                        message: '<%=rb.getString("ChengGong")%>'
                    });
                }else {
                    vm.$message({
                        type: 'warning',
                        message: '<%=rb.getString("ShiBai")%>'
                    });
                }				
                vm.closeFileSelect();
                vm.verifiedAccountImportShow = false;
            }else{
                vm.$message({
                    type: 'error',
                    message: res.message
                });
            }
            //修改已选择文件状态  
            var fileList = vm.$refs.verifiedAccountUpload.uploadFiles;
            fileList.forEach(function(file){
                file.status = 'ready';
            })
        },
        /**
        * 选择文件后，校验格式，并赋值页面显示 
        * @param file{object}   文件信息
        * @param fileList{Array}  文件列表
        */ 
        verifiedAccountFileChange(file,fileList){ 
            var vm = this,typeFlag;

            if(vm.importType == 'user' || vm.importType == 'block'){
                typeFlag = file.name.substr(file.name.lastIndexOf("."))  === '.csv';
            }else if(vm.importType == 'CA' || vm.importType == 'IPsec'){
                typeFlag = file.name.substr(file.name.lastIndexOf("."))  === '.crt';
            }else if(vm.importType == 'Key'){
                typeFlag = file.name.substr(file.name.lastIndexOf("."))  === '.key';
            }
            
            vm.verifiedAccountImportForm.errMassageShow = !typeFlag;
            if(typeFlag){
                vm.verifiedAccountImportFileName = file.name;
                vm.verifiedAccountImportFileParams.FileName = file.name;
            }else {
                vm.verifiedAccountImportFileName = '';
            }
           if(vm.importType == 'user' || vm.importType == 'block'){
                vm.verifiedAccountImportForm.errMassage = 'Only .csv is supported'
            }else if(vm.importType == 'CA' || vm.importType == 'IPsec'){
                vm.verifiedAccountImportForm.errMassage = 'Only .crt is supported'
            }else if(vm.importType == 'Key'){
                vm.verifiedAccountImportForm.errMassage = 'Only .key is supported'
            }
        },
        // 移除导入文件
        closeFileSelect(){
            var vm = this;
            vm.verifiedAccountImportFileName = '';
            vm.verifiedAccountImportFileParams.FileName = '';			
            vm.$refs.verifiedAccountUpload.clearFiles();
        },
        // 关闭导入弹出框
        closeImportParams(){
            var vm = this,
                params = {
                    errMassage:'Only .csv is supported',
                    errMassageShow:false
                };
            Object.assign(vm.verifiedAccountImportForm,params);		
            vm.verifiedAccountImportFileName = '';
            vm.verifiedAccountImportFileParams.FileName = '';
            vm.$refs.verifiedAccountUpload.clearFiles();
            vm.verifiedAccountImportShow = false;
        },
        // 导出用户列表
        exportVerifiedAccount(){
            var vm = this,
                urls = "${ctx}/egw/ipsec/exportUserList.action",
                params = {
                    egw_code: vm.egwCode,
                    timeZone:timeZone,
                    searchText: vm.queryVerifiedAccountParams.searchText
                };
            exportByForm(urls,params);
        },
        //下载模板
        verifiedAccountExportTemplate(){
            var vm = this,
                urls = '';
            if(vm.importType == 'user'){
                urls = '${ctx}/egw/ipsec/downloadImportWCGAAAUserTemplate.action';
            }else{
                urls = '${ctx}/egw/ipsec/downloadImportWCGUsimBlackListTemplate.action';
            }
            exportByForm(urls, {});
        },
        /*确定导入*/
        verifiedAccountUploadParams() {
            var vm = this;
            if(vm.verifiedAccountImportFileParams.FileName){
                vm.$refs.verifiedAccountUpload.submit();
            }else{
                vm.verifiedAccountImportForm.errMassageShow = true;
                vm.verifiedAccountImportForm.errMassage = '<%=rb.getString("QingXianXuanZeWenJian")%>'
            }
        },
        // 新增用户认证事件
        addVerifiedAccountClick(){
            var vm = this;
            vm.optVerifiedAccountType = 'add';
            vm.addVerifiedAccountClose();
            vm.blockListBoxShow = false;
            vm.addVerifiedAccountBoxShow = true
        },
        // 新增用户认证 提交
        addVerifiedAccountSubmit(){
            var vm =this,
                urls = '',
                params={
                    egwCode:vm.egwCode,
                    imsi:vm.addVerifiedAccountForm.imsi,
                    key:vm.addVerifiedAccountForm.key,
                    opc:vm.addVerifiedAccountForm.opc,
                    userSn:vm.addVerifiedAccountForm.userSn,
                    userMac:vm.addVerifiedAccountForm.userMac,
                    rcConstId:vm.addVerifiedAccountForm.rcConstId,
                    vendor:vm.addVerifiedAccountForm.vendor,
                    showAdditionalCol:vm.addVerifiedAccountForm.showAdditionalCol
                    
                };
            if(vm.optVerifiedAccountType == 'add'){
                urls = '${ctx}/egw/ipsec/addUser.action';
            }else{
                params.index = vm.verifiedAccountRowData.index;
                urls = '${ctx}/egw/ipsec/updateUser.action';
            }
            vm.$refs.addVerifiedAccountForm.validate(function(valid){
                if(valid){
                    axios.post(urls,stringify(params)).then(function(response){
                        var data = response.data;
                        if(data["success"]){
                            vm.$message({
                                message: '<%=rb.getString("MingLingYiXiaFa")%>',
                                type:'success',
                            })
                            vm.$refs.verifiedAccountTable.refresh();
                            vm.addVerifiedAccountClose();
                        }else{
                            vm.$message.error(data["message"])
                        }
                    }).catch(function(error){})
                }
            })

        },
        // 关闭新增用户认证
        addVerifiedAccountClose(){
            var vm = this,
                params={
                    imsi:'',
                    key:'',
                    opc:'',
                    userSn:'',
                    userMac:'',
                    rcConstId:'',
                    vendor:'',
                    showAdditionalCol:'0'
                };
            vm.addVerifiedAccountBoxShow = false;
            Object.assign(vm.addVerifiedAccountForm,params);
        },
        // 打开黑名单列表
        blockListClick(){
            var vm = this;
            vm.blockListBoxShow = true;
        },
        // USIM黑名单 模糊查询
        queryBlockList(val){
            var vm = this;
            vm.queryBlockListParams.searchText= val;
        },
        // 新增 USIM黑名单信息
        addBlockListClick(){
            var vm =this;
            vm.addBlockListDialogShow = true;
        },
        // 新增 USIM黑名单提交
        addBlockListSubmit(){
            var vm =this,
                urls = '${ctx}/egw/ipsec/addBlackList.action',
                params={
                    egwCode:vm.egwCode,
                    imsi:vm.addBlockListForm.imsi,
                };
            vm.$refs.addBlockListForm.validate(function(valid){
                if(valid){
                    axios.post(urls,stringify(params)).then(function(response){
                        var data = response.data;
                        if(data["success"]){
                            vm.$message({
                                message: '<%=rb.getString("MingLingYiXiaFa")%>',
                                type:'success',
                            })
                            vm.$refs.blockListTable.refresh();
                            vm.closeAddBlockList();
                        }else{
                            vm.$message.error(data["message"])
                        }
                    }).catch(function(error){})
                }
            })
        },
        // 关闭新增 USIM黑名单
        closeAddBlockList(){
            var vm = this,
                params={
                    imsi:'',
                };
            vm.addBlockListDialogShow = false;
            Object.assign(vm.addBlockListForm,params);
            vm.$refs.addBlockListForm.resetFields();
        },
        // 黑名单删除
        blockListDel(row,type){
            var vm = this,
                urls = '${ctx}/egw/ipsec/delBlackList.action',
                params={
                    egwCode:vm.egwCode,
                    indexList:''
                };
            if(type == 'del'){
                var confirmHint = '<%=rb.getString("QueRenShanChu")%>';
                params.indexList = row.index
            }else{
                var indexList = [],
                    confirmHint = '<%=rb.getString("ShanChuDangQianYeShuJu")%>';
                var blockList = vm.$refs.blockListTable.getData();
                blockList.map((item)=>{
                    indexList.push(item.index)
                })
                params.indexList = indexList.join(',');
            }
            vm.$confirm(confirmHint,'<%=rb.getString("QueRen")%>',{
                customClass:'warningConfirm',
                confirmButtonText:'<%=rb.getString("QueDing")%>',
                cancalButtonText:'<%=rb.getString("QuXiao")%>',
                type:'warning',
                closeOnClickModal:false
            }).then(()=>{
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
            }).catch(()=>{})
        },
        // 黑名单弹窗 关闭
        addBlockListClose(){
            var vm = this;
            vm.blockListBoxShow = false;
        },
        // 导出USIM黑名单
        exportBlockList(){
            var vm = this,
                urls = "${ctx}/egw/ipsec/exportUsimBlackList.action",
                params = {
                    egw_code: vm.egwCode,
                    timeZone:timeZone,
                    searchText: vm.queryBlockListParams.searchText
                };
            exportByForm(urls,params);
        },
        // 判断数据有无变化
        isObjectChange(obj1,obj2){
            var str1 = JSON.stringify(obj1);
            var str2 = JSON.stringify(obj2);
            if(str1 != str2){
                return true
            }
            return false
        },
        // 下发文件
        issueCertFile(){
            var vm = this,
                urls = '${ctx}/egw/ipsec/config/sendFile.action',
                params = {
                    egw_code:vm.egwCode,
                    serial_number:vm.egwSn,
                    ca_cert: vm.IPsecCertForm.ca_cert,
                    ipsec_cert:vm.IPsecCertForm.ipsec_cert,
                    private_key:vm.IPsecCertForm.private_key,
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
        // 更新证书
        updateCertFile(){
            var vm = this,
                urls = '${ctx}/egw/ipsec/config/updateCert.action',
                params = {
                    egw_code:vm.egwCode,
                    serial_number:vm.egwSn,
                    ca_cert: vm.IPsecCertForm.ca_cert,
                    ipsec_cert:vm.IPsecCertForm.ipsec_cert,
                    private_key:vm.IPsecCertForm.private_key,
                };
            if(vm.issueAndUpdateCertBtnDis)return
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
        // 认证用户表格 模糊查询
        queryVerifiedAccount(val){
            var vm = this;
            vm.queryVerifiedAccountParams.searchText= val;
        },
        // 认证用户表格选择
        verifiedAccountSelect(selection){
            var vm = this;
            vm.verifiedAccountSelection = selection
        },
        //  打开操作菜单
        verifiedAccountOptClick(row,evt){
            var vm =this;
            vm.verifiedAccountRowData = row;
            vm.verifiedAccountMenus = [
                {label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit CODE_EGW hidden",code:'mod'},
                {label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete CODE_EGW hidden",code:'del'},
                {label:'<%=rb.getString("YiDongDaoHeiMingDan")%>',cls:"el-icon el-icon-move CODE_EGW hidden",code:'move'},
            ];
            vm.$nextTick(function(){
                document.body.click();
                vm.$refs.verifiedAccountMenu.show(evt)
            })
        },
        /**
        * 菜单点击事件
        * @param ev{object}   行数据
        */ 
        verifiedAccountMenuClick(evt){
            var vm =this;
            var codes = {
                mod:this.verifiedAccountModify,	// 修改
                del:this.verifiedAccountDel,	// 删除
                move:this.verifiedAccountMove   // 移动到黑名单
            };
            if(codes[evt.code]){
                codes[evt.code]('alone');
            }
        },
        // 菜单关闭
        hideverifiedAccountMenus(){
            this.$refs.verifiedAccountMenu.hide();
        },
        // 认证用户 修改
        verifiedAccountModify(){
            var vm = this;
            vm.optVerifiedAccountType = 'edit';
            vm.blockListBoxShow = false;
            vm.addVerifiedAccountBoxShow = true;
            Object.assign(vm.addVerifiedAccountForm,vm.verifiedAccountRowData);
        },
        // 认证用户 删除
        verifiedAccountDel(type){
            var vm = this,
                urls = '${ctx}/egw/ipsec/delUserList.action',
                params={
                    egwCode:vm.egwCode,
                    indexList:''
                };
            if(type == 'alone'){
                params.indexList = vm.verifiedAccountRowData.index
            }else{
                var indexList=[];
                vm.verifiedAccountSelection.map((item,index) => {
                    indexList.push(item.index)
                })
                params.indexList = indexList.join(',');
            }
            vm.$confirm('<%=rb.getString("QueRenShanChu")%>','<%=rb.getString("QueRen")%>',{
                confirmButtonText:'<%=rb.getString("QueDing")%>',
                cancalButtonText:'<%=rb.getString("QuXiao")%>',
                type:'warning'
            }).then(()=>{
                axios.post(urls,stringify(params)).then(res=>{
                    var data = res.data;
                    if(data["success"]){
                        vm.$message({
                            message: '<%=rb.getString("MingLingYiXiaFa")%>',
                            type:'success',
                        });
                        vm.$refs.verifiedAccountTable.refresh();
                        vm.$refs.blockListTable.refresh();
                    }else{
                        vm.$message.error(data["message"])
                    }
                })
            }).catch(()=>{})
        },
        // 认证用户 移动到黑名单
        verifiedAccountMove(type){
            var vm = this,
                urls = '${ctx}/egw/ipsec/userMoveToBlackList.action',
                params={
                    egwCode:vm.egwCode,
                    indexList:'',
                    imsiList:''
                };
            if(type == 'alone'){
                params.imsiList = vm.verifiedAccountRowData.imsi
                params.indexList = vm.verifiedAccountRowData.index
            }else{
                var indexList=[],imsiList=[];
                vm.verifiedAccountSelection.map((item,index) => {
                    indexList.push(item.index);
                    imsiList.push(item.imsi);
                })
                params.indexList = indexList.join(',');
                params.imsiList = imsiList.join(',');
            }
            vm.$confirm('<%=rb.getString("QueRenJiaRuHeiMingDan")%>','<%=rb.getString("QueRen")%>',{
                confirmButtonText:'<%=rb.getString("QueDing")%>',
                cancalButtonText:'<%=rb.getString("QuXiao")%>',
                type:'warning'
            }).then(()=>{
                axios.post(urls,stringify(params)).then(res=>{
                    var data = res.data;
                    if(data["success"]){
                        vm.$message({
                            message: '<%=rb.getString("MingLingYiXiaFa")%>',
                            type:'success',
                        });
                        vm.$refs.verifiedAccountTable.refresh();
                        vm.$refs.blockListTable.refresh();
                    }else{
                        vm.$message.error(data["message"])
                    }
                })
            }).catch(()=>{})
            
        },
        // 关闭配置页面
        closeLinkSetting(){
            var vm = this;
            egwMonitor.$refs.egwSettingPage.hide();
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
                    refreshType: 'SECURITYGATEWAY'
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
