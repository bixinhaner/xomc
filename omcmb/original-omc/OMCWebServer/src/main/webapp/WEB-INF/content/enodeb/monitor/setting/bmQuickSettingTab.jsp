<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
    #cellSettingPage .quickSettingAdd {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 4px; 
        border: 1px solid #dcdcdc;
        border-radius: 4px;
        padding: 0 10px;
        width: fit-content; 
        background: #fff;
        cursor: pointer;
        margin-top: 34px;
        height: 24px;
        background: rgba(var(--main-color-rgba1), 0.1);
        border-color: var(--main-color); 
    }
    #cellSettingPage .quickSettingAdd p {
        margin: 0;
        padding: 0;
        font-size: 12px;
        line-height: 1;
        color: var(--main-color);
    }
    #cellSettingPage .itemTitleCls {
		font-weight: bold;
		position: relative;
		padding-left: 15px;
		margin-bottom: 4px;
        font-size: 12px;
        color: rgba( 0, 0, 0, 0.8)
	}
	#cellSettingPage .itemTitleCls:before {
		content: ' ';
		width: 6px;
		height: 6px;
		background: #000;
		border-radius: 6px;
		display: inline-block;
		position: absolute;
		top: 8px;
		left: 0px;
	}
    .cellConfigDialog .el-form {
        display: flex;
        flex-wrap: wrap;
        gap: 0 2%;
        width: 100%;
    }
    .cellConfigDialog .el-form-item {
        flex: 1 1 48%;
        min-width: 220px;
        box-sizing: border-box;
        margin-bottom: 14px !important;
        margin-left: 0 !important;
        max-width: 100%;
    }
    .cellConfigDialog .el-form-item__label {
        display: block;
        width: 100%;
        white-space: normal;
        margin-bottom: 0;
        line-height: 26px !important;
        margin-left: unset;
        font-size: 14px !important;
        color: rgba(0, 0, 0, 0.8);
    }
    .cellConfigDialog .el-input-group__append {
        padding: 0 10px !important;
    }
    .cellConfigDialog P {
        width: 100%;
    }
    .cellConfigDialog .allowMoreInputBoxCls {
		position: relative;
		flex: 1;
	}
	.cellConfigDialog .allowMoreInputBoxCls .allowMoreInputHeadCls {
		margin-bottom: 5px;
	}
	.cellConfigDialog .allowMoreInputBoxCls .allowMoreInputHeadCls .allowMoreInputTitleCls {
		font-size: 14px;
		color: rgba(0, 0, 0, 0.8);
	}
	.cellConfigDialog .allowMoreInputBoxCls .allowMoreInputHeadCls .allowMoreInputTipsCls {
		font-size: 14px;
		color: rgba(0, 0, 0, 0.32);
	}
	.cellConfigDialog .allowMoreInputBoxCls .allowMoreInputContentCls {
		border: 1px solid #DFE2EE;
		width: 80%;
		min-width: 600px;
		min-height: 78px;
		padding: 10px;
		border-radius: 4px;
		box-sizing: border-box;
	}
	.cellConfigDialog .allowMoreInputBoxCls .allowMoreInputFieldCls {
		display: flex;
	}
	.cellConfigDialog .allowMoreInputBoxCls .allowMoreInputFieldCls .el-input {
		width: 240px;
	}
	.cellConfigDialog .allowMoreInputBoxCls .allowMoreInputFieldCls .allowMoreInputAddBtnCls {
		height: 26px;
		width: 56px;
		display: flex;
		align-items: center;
		justify-content: center;
		color: var(--main-color);
		border: 1px solid var(--main-color);
		background: rgba(var(--main-color-rgba1),0.1);
		border-radius: 4px;
		box-sizing: border-box;
		margin-left: 10px;
		cursor: pointer;
	}
	.cellConfigDialog .allowMoreInputBoxCls .allowMoreInputFieldCls .allowMoreInputAddTipCls {
		margin-left: 10px;
	}
	.cellConfigDialog .allowMoreInputBoxCls .allowMoreInputFieldCls .allowMoreInputAddBtnCls .el-icon::before {
		font-size: 14px;
		color: var(--main-color);
	}
	.cellConfigDialog .allowMoreInputBoxCls .allowMoreInputFieldCls .allowMoreInputAddBtnCls span:nth-child(2) {
		margin: 0px 3px;
	}
	.cellConfigDialog .allowMoreInputBoxCls .allowMoreInputParamsCls {
		display: flex;
		flex-wrap: wrap;
		width: 100%;
		margin-top: 10px;
	}
	.cellConfigDialog .allowMoreInputBoxCls .allowMoreInputParamsItemCls {
		height: 26px;
		display: flex;
		align-items: center;
		border: 1px solid #DFE2EE;
		border-radius: 4px;
		box-sizing: border-box;
		padding: 4px 10px 0;
		margin-right: 10px;
		background: #F8F8FD;
	}
	.cellConfigDialog .allowMoreInputBoxCls .allowMoreInputParamsItemCls span:nth-child(1) {
		display: inline-block;
		max-width: 520px;
		overflow: hidden;
		text-overflow: ellipsis;
		height: 100%;
	}
	.cellConfigDialog .allowMoreInputBoxCls .allowMoreInputParamsItemCls span:nth-child(2) {
		margin-left: 10px;
	}
	.cellConfigDialog .allowMoreInputBoxCls .allowMoreInputParamsItemCls .el-icon-close {
		font-size: unset;
		position: unset;
		top: unset;
		right: unset;
	}
	.cellConfigDialog .allowMoreInputBoxCls .allowMoreInputParamsItemCls .el-icon-close::before {
		font-size: 12px;
		color: #7A7992;
	}
	.cellConfigDialog .allowMoreInputBoxCls .allowMoreInputFootCls {
		height: 18px;
	}
	.cellConfigDialog .allowMoreInputBoxCls .allowMoreInputFootCls .inputErrorBoxCls {
		color:red;
		font-size:10px;
	}
    #cellSettingPage .cellConfigCls .el-form-item__content .el-form-item__error {
        width: unset !important;
        left: unset !important;
        top: unset !important;
    }
    .cellConfigDialog .mmeSelect1 .el-form-item__content {
        display: flex;
    }
    .cellConfigDialog .mmeSelect1 .el-input,
    .cellConfigDialog .mmeSelect2 .el-input__inner{
		width: 80px !important;
	}
    .cellConfigDialog .mmeSelect .el-input,
    .cellConfigDialog .mmeSelect .el-input__inner{
		width: 95px !important;
	}
</style>
<div id='cellSettingPage' style='height: 100%;overflow: auto;background: #fff;min-width: 900px;'>
	<el-form ref="cellConfigForm" :model="cellConfigForm" :rules="cellConfigRules" 
		label-position="top" style='width:100%;height:100%;' inline>
		<el-collapse v-model="activeNames">
			<el-collapse-item name="cellConfig">
				<template slot='title'>
					<p style="display:inline-block;margin-left:40px;">
						<span class="title-icon" style="vertical-align:sub"></span>
						<span style="font-size:14px;font-weight:bold"><%= rb.getString("XinXiaoQu")%></span>
					</p>
				</template>
				<div style="display: flex;">
                    <el-form-item label='<%= rb.getString("XinXiaoQuPeiZhi")%>' class="cellConfigCls" style="width: 205px; margin-left: 0;">
                        <el-select style='vertical-align:bottom;margin-left:-3px;' v-model='cellConfigAdd'>
                            <el-option v-for="item in cellOptions" v-if="item.enable == '0'" 
                                :label="item.text" 
                                :value="item.index">
                            </el-option>
                        </el-select>
                    </el-form-item>
                    <div class="quickSettingAdd" @click="addCellConfig">
                        <p class="el-icon el-icon-plus"></p>
                        <p><%= rb.getString("TianJia")%></p>
                    </div>
                </div>

                <p class='itemTitleCls'><%= rb.getString("LTEXiaoQuSheZhi")%></p>
				<div style="height:158px;width:96%;border:1px solid #F3F3F3; margin-bottom: 20px;">
					<el-ctable ref="lteCtable" :data="lteCellTbList"
                        :pagination="false" :rownumber="false" >
						<el-table-column width="80"class-name="no-text-tips">
							<template slot-scope="scope">
								<!-- 常规操作 -->
								<span class="el-icon el-icon-operation-edit" @click="editLteCellClick(scope.row)"  style="cursor: pointer;"></span>
								<span class="el-icon el-icon-operation-delete" @click="delLteCellClick(scope.row)" style="margin-left:10px;"></span>
							</template>
						</el-table-column>
                        <el-table-column label="Index" prop="lteCellIndex" min-width="60"></el-table-column>
						<el-table-column label='<%=rb.getString("ShiFouJiHuo") %>' prop="cellStatus" min-width="120" show-overflow-tooltip>
                            <template slot-scope="scope">
                                <div v-if="scope.row.cellStatus == '0'" style="color:#FF4614;"><%= rb.getString("QuJiHuo")%></div>
                                <div v-else-if="scope.row.cellStatus == '1'"><%= rb.getString("JiHuo")%></div>
                                <div v-else></div>
                            </template>
                        </el-table-column>
						<el-table-column label='<%=rb.getString("ShePinKaiGuanZhuangTai")%>' prop="rfStatus" min-width="120" show-overflow-tooltip>
                            <template slot-scope="scope">
                                <div v-if="scope.row.rfStatus == '0'" style="color:#FF4614;"><%= rb.getString("Guan")%></div>
                                <div v-else-if="scope.row.rfStatus == '1'"><%= rb.getString("Kai")%></div>
                                 <div v-else></div>
                            </template>
                        </el-table-column>
						<el-table-column label='<%=rb.getString("DaiKuan")%>' prop="bandwidth" min-width="140" :formatter="bandwidthFmtOne" show-overflow-tooltip></el-table-column>
						
						<el-table-column label="ECI" prop="eci" min-width="120" show-overflow-tooltip></el-table-column>
						<el-table-column label='<%=rb.getString("PinLvHeZi")%>' prop="earfcn" min-width="140" show-overflow-tooltip>
                            <template slot-scope="scope">
                                <!--根据 band 和 earfcn 转换频率显示-->
                                <span v-if="scope.row.earfcn && scope.row.band">{{ formatConversion(scope.row.earfcn, scope.row.band) }}</span>
                                <span v-else></span>
                            </template>
                        </el-table-column>
						<el-table-column label='<%=rb.getString("ENBPinDian")%>' prop="earfcn" min-width="100" show-overflow-tooltip></el-table-column>
						<el-table-column label='<%=rb.getString("CPETxPower")%>' prop="transmissionPower" min-width="120" show-overflow-tooltip>
                            <template slot-scope="scope">
                                <div>{{scope.row.transmissionPowerFirst}}*{{scope.row.transmissionPower}}dBm</div>
                            </template>
                        </el-table-column>
						<el-table-column label="PCI" prop="pci" min-width="100" show-overflow-tooltip></el-table-column>
						<el-table-column label="CPRI ID" prop="lteCellIndex" min-width="100" show-overflow-tooltip>
                            <!--索引+1-->
                            <template slot-scope="scope">
                                <div>{{Number(scope.row.lteCellIndex) + 1}}</div>
                            </template>
                        </el-table-column>
						<el-table-column label='<%=rb.getString("LuYouSuoYin")%>' prop="routeIndex" min-width="100" show-overflow-tooltip></el-table-column>						
					</el-ctable>
				</div>

                <p class='itemTitleCls'><%=rb.getString("GSMXiaoQuSheZhi")%></p>
				<div style="height:158px;width:96%;border:1px solid #F3F3F3;">
					<el-ctable ref="gsmCtable" :data="gsmCellTbList" :rownumber="false" :pagination="false">
						<el-table-column width="80" class-name="no-text-tips">
							<template slot-scope="scope">
								<span class="el-icon el-icon-operation-edit" @click="editGsmCellClick(scope.row)" style="cursor: pointer;"></span>
								<span class="el-icon el-icon-operation-delete" @click="delGsmCell(scope.row)" style="margin-left:10px;"></span>
							</template>
						</el-table-column>
						<el-table-column label="Index" prop="gsmCellIndex" min-width="60"></el-table-column>
						<el-table-column label='<%=rb.getString("ShiFouJiHuo") %>' prop="cellStatus" min-width="120" show-overflow-tooltip>
                            <template slot-scope="scope">
                                <div v-if="scope.row.cellStatus == '0'" style="color:#FF4614;"><%= rb.getString("QuJiHuo")%></div>
                                <div v-else-if="scope.row.cellStatus == '1'"><%= rb.getString("JiHuo")%></div>
                                <div v-else></div>
                            </template>
                        </el-table-column>
						<el-table-column label='<%=rb.getString("ShePinKaiGuanZhuangTai")%>' prop="rfStatus" min-width="120" show-overflow-tooltip>
                            <template slot-scope="scope">
                                <div v-if="scope.row.rfStatus == '0'" style="color:#FF4614;"><%= rb.getString("Guan")%></div>
                                <div v-else-if="scope.row.rfStatus == '1'"><%= rb.getString("Kai")%></div>
                                 <div v-else></div>
                            </template>
                        </el-table-column>
						<el-table-column label='<%=rb.getString("AbisLianLuZhuangTai")%>' prop="abisLinkStatus" min-width="140" show-overflow-tooltip>
                            <template slot-scope="scope">
                                <div v-if="scope.row.abisLinkStatus == '255'"><%=rb.getString("IpsecWeiLianJie")%></div>
                                <div v-else-if="scope.row.abisLinkStatus == '0' || scope.row.abisLinkStatus == '1'"><%= rb.getString("LianJieZhengChang")%></div>
                                <div v-else></div>
                            </template>
                        </el-table-column>
						<el-table-column label='<%=rb.getString("YongHuSheBeiZhuangTai")%>' prop="ueConnections" min-width="140" show-overflow-tooltip></el-table-column>
						<el-table-column label='<%=rb.getString("XIAOQUID")%>' prop="gsmCellID" min-width="100" show-overflow-tooltip></el-table-column>
						<el-table-column label='<%=rb.getString("WeiZhiQuXinXi")%>' prop="lac" min-width="100" show-overflow-tooltip></el-table-column>
						<el-table-column label="PLMN" prop="plmn" min-width="100" show-overflow-tooltip>
                            <template slot-scope="scope">
                                <!--两个字段的组合 plmn + plmnSecond-->
                                <div>{{scope.row.plmn}}{{scope.row.plmnSecond}}</div>
                            </template>
                        </el-table-column>
						<el-table-column label='<%=rb.getString("JiZhanShiBieMa")%>' prop="gsmBSIC" min-width="100" show-overflow-tooltip></el-table-column>
						<el-table-column label="CPRI ID" prop="gsmCellIndex" min-width="100" show-overflow-tooltip>
                            <!--索引+1-->
                            <template slot-scope="scope">
                                <div>{{Number(scope.row.gsmCellIndex) + 1}}</div>
                            </template>
                        </el-table-column>
						<el-table-column label='<%=rb.getString("LuYouSuoYin")%>' prop="routeIndex" min-width="100" show-overflow-tooltip></el-table-column>						
					</el-ctable>
				</div>
			</el-collapse-item>
		</el-collapse>
	</el-form>
    <!--添加小区配置确认对话框-->
    <el-dialog class="cellConfigDialog" top="25vh" width="22%" 
        title="Confirmation" 
        :visible.sync="cellConfigShow" 
        :close-on-click-modal="false" 
        :modal-append-to-body="false" 
        @close="closeCellConfigDialog">
        <div>
			<p class="commonSize14"><%=rb.getString("XinQueRenXinJianRenWu")%></p>
			<p style="font-size: 14px; color: #9E9E9E; padding-top: 10px;"><%=rb.getString("PeiZhiJiangZaiChongQiHouShengXiao")%></p>
		</div>
        
        <div slot="footer" class="importFooter" style="text-align: right;">
            <el-button type="primary" @click="addCellConfigSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="closeCellConfigDialog"><%=rb.getString("QuXiao")%></el-button>
        </div>
    </el-dialog>
	<!--edit lte cell-->
    <el-dialog class="cellConfigDialog" top="10vh" width="840px" 
        title="Modify" 
        :visible.sync="editLteCellShow" 
        :close-on-click-modal="false" 
        :modal-append-to-body="false" 
        @close="editLteCellClose">
        <el-form ref="editLteCellForm" :model="editLteCellForm" :rules="editLteCellRules" label-width="200px" label-position="top">
            <el-form-item label='<%=rb.getString("LuYouSuoYin")%>' prop='routeIndex'>
                <el-input v-model.trim='editLteCellForm.routeIndex' disabled></el-input>
            </el-form-item>
            <el-form-item label="Band" prop='band'>
                <el-select v-model="editLteCellForm.band" @change="onBandChange"> 
                    <el-option label='3' value='3'></el-option>
                    <el-option label='8' value='8'></el-option>
                    <el-option label='20' value='20'></el-option>
                    <el-option label='28' value='28'></el-option>
                </el-select>
            </el-form-item>
            <el-form-item label='<%=rb.getString("DaiKuan")%>' prop="bandwidth">
                <el-select v-model="editLteCellForm.bandwidth" @change="onBandwidthChange">
                    <el-option label='5MHz' value='25'></el-option>
                    <el-option label='10MHz' value='50'></el-option>
                    <el-option label='15MHz' value='75'></el-option>
                    <el-option label='20MHz' value='100'></el-option>
                </el-select>
            </el-form-item>
            
            <el-form-item label='<%=rb.getString("ENBPinDian")%>' prop='earfcn' class='validate-item'>
                <el-input v-model.trim='editLteCellForm.earfcn' @input="onEarfcnInput">
                    <template slot="append">
                        <span v-if="currentEarfcnRange"><%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%>: {{currentEarfcnRange[0]}}~{{currentEarfcnRange[1]}}</span>
                        <span v-else><%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%>: 1~65535</span>
                    </template>
                </el-input>
            </el-form-item>
            <el-form-item label='<%=rb.getString("PinLvHeZi")%>'>
                <el-input v-model='lteFrequencyValue' disabled></el-input>
            </el-form-item>
            <el-form-item label="PCI" prop='pci' class='validate-item'>
                <el-input v-model.trim='editLteCellForm.pci'>
                    <template slot="append"><%=rb.getString("FanWei")%>:0~503</template>
                </el-input>
            </el-form-item>
            <el-form-item label="ECI(ECI=eNB_ID*256+Cell_ID)" prop='eci'>
                <el-input v-model.trim='editLteCellForm.eci' disabled></el-input>
            </el-form-item>
            <!--根据eci 进行计算得出-->
            <el-form-item class='validate-item'>
                <template slot="label">
                    <span style="color: #FF4614;">*</span><%=rb.getString("XIAOQUID")%>
                </template>
                <el-input v-model='lteCellIdValue' disabled>
                    <template slot="append"><%=rb.getString("FanWei")%>:0~255 <%=rb.getString("ZhengXing")%></template>
                </el-input>
            </el-form-item>
            <el-form-item label='<%=rb.getString("ShePinKaiGuanZhuangTai")%>' prop="rfStatus">
                <el-select v-model="editLteCellForm.rfStatus">
                    <el-option label='<%= rb.getString("Guan")%>' value='0'></el-option>
                    <el-option label='<%= rb.getString("Kai")%>' value='1'></el-option>
                </el-select>
            </el-form-item>
            <el-form-item label='<%=rb.getString("ChuanShuGongLv")%>' prop="transmissionPowerFirst" class='mmeSelect1'>
                <el-select v-model="editLteCellForm.transmissionPowerFirst">
                    <el-option label='2' value='2'></el-option>
                    <el-option label='4' value='4'></el-option>
                </el-select>
                <span style='margin:0 10px;'>X</span>
                <el-form-item label="" prop="transmissionPower">
                    <el-select v-model="editLteCellForm.transmissionPower" class='mmeSelect'>
                        <el-option v-for="item in txfPowerList" :key="item.value" :label="item.label" :value="item.value"></el-option>
                    </el-select>
                </el-form-item>
            </el-form-item>
            <div class="allowMoreInputBoxCls">
                <div class="allowMoreInputHeadCls">
                    <span class="allowMoreInputTitleCls">PLMN</span>
                </div>
                <div class="allowMoreInputContentCls">
                    <div class="allowMoreInputFieldCls">
                        <el-input v-model="defaultPlmn.plmn"></el-input>
                        <div v-if="defaultPlmnList.length < 6" class="allowMoreInputAddBtnCls" @click="addDefaultPlmn">
                            <span class="el-icon el-icon-plus"></span>
                            <span>Add</span>
                        </div>
                    </div>
                    <div class="allowMoreInputParamsCls">
                        <div v-for="item in defaultPlmnList" class="allowMoreInputParamsItemCls">
                            <span style="font-size: 12px;">{{item}}</span>
                            <span class="el-icon el-icon-close" style="margin-left: 10px; margin-top: -3px;" @click="defaultPlmnDel(item)"></span>
                        </div>
                    </div>
                </div>
                <div class="allowMoreInputFootCls">
                    <p class="inputErrorBoxCls">{{defaultPlmn.defaultPlmnErrorMessage}}</p>
                </div>
                <div style="font-size: 12px;color: #909399"><%=rb.getString("PLMNPeiZhiTiShi")%></div>
                <el-form-item prop='plmn' style="display:none;" label="" label-width="0px">
                    <el-input v-model='editLteCellForm.plmn'></el-input>
                </el-form-item>
            </div>
        </el-form>
        <div slot="footer" class="importFooter">
            <el-button type="primary" @click="editLteCellSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="editLteCellClose"><%=rb.getString("QuXiao")%></el-button>
        </div>
    </el-dialog>

    <!--修改 gsm cell-->
    <el-dialog class="cellConfigDialog" top="10vh" width="840px" 
        title="Modify" 
        :visible.sync="editGsmCellShow" 
        :close-on-click-modal="false" 
        :modal-append-to-body="false" 
        @close="editGsmCellClose">
        <el-form ref="editGsmCellForm" :model='editGsmCellForm' :rules='editGsmCellRules' label-width="200px" label-position="top">
            <p class="commonText14" style="padding-bottom: 10px;">Air Interface Settings</p>
            <el-form-item label="LAC" prop='lac'>
                <el-input v-model.trim='editGsmCellForm.lac' disabled></el-input>
            </el-form-item>
            <el-form-item label="ARFCN" prop='gsmArfcn'>
                <el-input v-model.trim='editGsmCellForm.gsmArfcn' disabled></el-input>
            </el-form-item>
            <el-form-item label='<%=rb.getString("PinLvHeZi")%>'>
                <el-input v-model='gsmFrequencyValue' disabled></el-input>
            </el-form-item>
            <el-form-item label='<%=rb.getString("ShiFouJiHuo")%>' prop="cellStatus">
                <el-select v-model="editGsmCellForm.cellStatus">
                    <el-option label='<%= rb.getString("QuJiHuo")%>' value='0'></el-option>
                    <el-option label='<%= rb.getString("JiHuo")%>' value='1'></el-option>
                </el-select>
            </el-form-item>
            <el-form-item label='<%=rb.getString("ChuanShuGongLv")%>' prop="transmissionPower">
                <el-select v-model="editGsmCellForm.transmissionPower">
                    <el-option v-for="item in txfPowerList" :key="item.value" :label="item.label" :value="item.value"></el-option>
                </el-select>
            </el-form-item>
            <el-form-item label='<%=rb.getString("LuYouSuoYin")%>' prop='routeIndex'>
                <el-input v-model.trim='editGsmCellForm.routeIndex' disabled></el-input>
            </el-form-item>

            <p class="commonText14" style="padding-bottom: 10px;"><%=rb.getString("GenZongBianHao")%>Abis Settings</p>
            <el-form-item label="" prop='ipaAndUnitId' class='validate-item' v-if="false">
                <el-input v-model.trim='editGsmCellForm.ipaAndUnitId'>
                    <template slot="append"><%=rb.getString("FanWei")%>:0-65534 <%=rb.getString("ZhengXing")%></template>
                </el-input>
            </el-form-item> 

            <el-form-item class='validate-item'>
                <template slot="label">
                    <span style="color: #FF4614;">*</span> ipa
                </template>
                <el-input v-model.trim='ipaValue' @input="validateIpa(0, 65534)"></el-input>
                <span v-if="ipaValueOk" style="margin-left:8px; color: #909399;"><%=rb.getString("FanWei")%>:0-65534 <%=rb.getString("ZhengXing")%></span>
                <span v-if="ipaValueError" style="color: #FF4614; margin-left:8px;"><%=rb.getString("FanWei")%>:0-65534 <%=rb.getString("ZhengXing")%></span>
            </el-form-item>
            <el-form-item class='validate-item'>
                <template slot="label">
                    <span style="color: #FF4614;">*</span><%=rb.getString("DanYuanID")%>
                </template>
                <el-input v-model.trim='unitIdValue' @input="validateUnitId(0, 255)"></el-input>
                <span v-if="unitIdOk" style="margin-left:8px; color: #909399;"><%=rb.getString("FanWei")%>:0-255 <%=rb.getString("ZhengXing")%></span>
                <span v-if="unitIdError" style="color: #FF4614; margin-left:8px;"><%=rb.getString("FanWei")%>:0-255 <%=rb.getString("ZhengXing")%></span>
            </el-form-item>
            <el-form-item label='<%=rb.getString("DuiDuanIP")%>' prop='remoteIp' class='validate-item'>
                <el-input v-model.trim='editGsmCellForm.remoteIp'>
                    <template slot="append"><%=rb.getString("ZhiZhiChiIPV4DiZhi")%></template>
                </el-input>
            </el-form-item>
            <el-form-item label='<%=rb.getString("YuanChengIPBeiFei")%>' prop='remoteIpBackup' class='validate-item'>
                <el-input v-model.trim='editGsmCellForm.remoteIpBackup'>
                    <template slot="append"><%=rb.getString("ZhiZhiChiIPV4DiZhi")%></template>
                </el-input>
            </el-form-item>

            <p class="commonText14" style="padding-bottom: 10px;"><%=rb.getString("TraceLog")%></p>
            <el-form-item label="CellDt" prop='cellDt'>
                <el-select v-model="editGsmCellForm.cellDt">
                    <el-option label='<%= rb.getString("Guan")%>' value='0'></el-option>
                    <el-option label='<%= rb.getString("Kai")%>' value='1'></el-option>
                </el-select>
            </el-form-item>
            <el-form-item label='<%=rb.getString("GenZongXiaoXiLeiXing")%>' prop='traceMessageType'>
               <el-select v-model="editGsmCellForm.traceMessageType">
                    <el-option label='<%=rb.getString("SheZhiTTIGenZongBianHao")%>' value='1'></el-option>
                    <el-option label='<%=rb.getString("BuHuoIQRiZhi")%>' value='3'></el-option>
                    <el-option label='<%=rb.getString("XianShiTTIGenZongBianHao")%>' value='9'></el-option>
                </el-select>
            </el-form-item>
            <el-form-item label='<%=rb.getString("CunGenZhi")%>' prop='stubValue' class='validate-item'>
                <el-input v-model.trim='editGsmCellForm.stubValue'>
                    <template slot="append"><%=rb.getString("FanWei")%>:0-65535 <%=rb.getString("ZhengXing")%></template>
                </el-input>
            </el-form-item>
            <el-form-item label='<%=rb.getString("GenZongBianHao")%>' prop='traceNum' class='validate-item'>
                <el-input v-model.trim='editGsmCellForm.traceNum' maxlength="99">
                    <template slot="append"><%=rb.getString("FanWei")%>:0-99 <%=rb.getString("ZiFuFuShu")%></template>
                </el-input>
            </el-form-item>
        </el-form>
        <div slot="footer" class="importFooter">
            <el-button type="primary" @click="editGsmCellSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="editGsmCellClose"><%=rb.getString("QuXiao")%></el-button>
        </div>
    </el-dialog>
</div>
<script>
	var lteVm = new Vue({
		el:'#cellSettingPage',
		data(){
			var vm = this,
                validateRange = (rule,value,callback)=>{
                    var min = rule.min,
                        max = rule.max,
                        reg = /^(0|[1-9][0-9]*)$/; 

                    if(value == '' || !reg.test(value) || value < min || value > max){
                        callback(new Error('format error'))
                    }else{
                        callback();
                    }
                },
                // EARFCN 范围验证
                validateEarfcnRange = (rule, value, callback) => {
                    var vm = this,
                        min = vm.currentEarfcnRange ? vm.currentEarfcnRange[0] : 1,
                        max = vm.currentEarfcnRange ? vm.currentEarfcnRange[1] : 65535,
                        reg = /^-?\d+$/;

                    if (value == '' || !reg.test(value) || value < min || value > max) {
                        callback(new Error('Integer,Range: ' + min + ' - ' + max));
                    } else {
                        // 更新频率显示
                        vm.lteFrequencyValue = vm.formatConversion(value, vm.editLteCellForm.band);
                        callback();
                    }
                },
                validateIp = (rule,value,callback)=>{
                    if(value == '' || value == null){
                        callback(new Error('<%=rb.getString("IPGeShiBuDui")%>'))
                    }else {
                        if(vm.isValidIP(value)){
                            callback();
                        }else{
                            callback(new Error('<%=rb.getString("IPGeShiBuDui")%>'))
                        }
                    }
                };
			return {
                activeNames: ['cellConfig'],	
                cellConfigAdd: '', // cell config 选择值
                cellConfigForm: {},
                cellConfigRules: {},
                lteRowData: {}, //重新组合后的 lte cell table row data
                gsmRowData: {}, //重新组合后的 lte cell table row data
                originLteRowData: {}, //原始的 lte cell row data
                originGsmRowData: {}, //原始的 gsm cell row data

                codeList: [],  				
				codeTableList: [],
				
				//添加小区的配置			
                cellConfigShow: false,
				smallCellCode:'',
                //lte cell
                editLteCellShow: false,
                editLteCellForm: {
                    enable: '', //cell enable
					lteCellIndex: '',//index
                    cellStatus: '',//cell status
                    rfStatus: '',//rf status
                    band: '',//band
                    bandwidth: '',//bandwidth
                    eci: '',//eci
                    earfcn: '',//earfcn
                    transmissionPowerFirst: '',// 1st part of transmission power
                    transmissionPower: '',// 2nd part of transmission power
                    pci: '',//pci
                    routeIndex: '',//route index
                    plmn: '' //plmn
                },
                //editLteCellForm 无需配置参数 Frequency, Cell ID, CPRI ID

                lteCellIdValue: '',
                lteFrequencyValue: '',
                editLteCellRules: {
                    pci: [ { required: true, validator: validateRange, min: 0, max: 503 } ],
                    earfcn: [ { required: true, validator: validateEarfcnRange } ],
                },
                
                defaultPlmn:{
                    plmn:'',
                    defaultPlmnErrorMessage:'',
                },
                defaultPlmnList: [],
                //gsm cell
                editGsmCellShow: false,
                editGsmCellForm: {
                    enable: '',
					gsmCellIndex: '',
                    cellStatus: '',
                    rfStatus: '',
                    abisLinkStatus: '',
                    ueConnections: '',
                    gsmCellID: '',
                    lac: '', 
                    plmn: '',
                    plmnSecond: '',
                    gsmBSIC: '',
                    routeIndex: '',
                    gsmArfcn: '',
                    transmissionPower: '',
                    ipaAndUnitId: '',
                    remoteIp: '',
                    remoteIpBackup: '',
                    cellDt: '',
                    traceMessageType: '',
                    stubValue: '',
                    traceNum: ''
                },
                //editGsmCellForm 无需配置参数 Frequency, CPRI ID
                ipaValue: '',
                unitIdValue: '',
                gsmFrequencyValue: '',
                editGsmCellRules: {
                    remoteIp: [ { required: true, validator: validateIp } ],
                    remoteIpBackup: [ { required: true, validator: validateIp } ],
                    stubValue: [ { required: true, validator: validateRange, min: 0, max: 65535} ],
                },
                txfPowerList: [],  
                ipaValueOk: true,
                ipaValueError: false,
                unitIdOk: true,
                unitIdError: false,
                casts:{
                    //lte cell 1  第一组数据
                    '0':'lteCellIndex0', 
                    '17C1B28826341D21A9695446B54528D5':'enable0',
                    '53550CDE5EFC208C01C6DB23CF99A242':'cellStatus0',
                    'C16F053DAD76488940AE36DAFAB05DD8':'rfStatus0',
                    '19E793A240D2E04EFB520BA1415C6C8E':'band0',    
                    '3B9BCB20DA630BB909F37807CC3A2CA7':'bandwidth0',
                    'B471C10C1E3882E9E404B5D8E4E5C79F':'eci0',
                    '6BFB6D72A2C84375FEC722F36E843A50':'earfcn0',
                    '52B7BDBB5D1B07AFBEAA958A0BE841B9':'pci0',
                    '702F9993EBFF3B9544220ABF24349D71':'routeIndex0',
                    'CE6801DEF400E2A1BBFD60CBD23B5380':'plmn0',
                    'C488255FC869387264FAB93612C70B49': 'transmissionPowerFirst0',
                    'AE2D44FB529D4CCBD6B198D5274355D0': 'transmissionPower0',
                    //lte cell 2 第二组数据
                    '1':'lteCellIndex1', 
                    '97442F3B1E08A9B0616C9813EC17E45C':'enable1',
                    '2B46400422F9D5E7F94EEE9B7F373607':'cellStatus1',
                    'D22497BF63292C42A05EBB10039AD568':'rfStatus1',
                    '4D601E9CE78DE9173728E2E2A100371E':'band1',
                    'B7603FA1E0908A6C9B93395C80B27F98':'bandwidth1',
                    '7593BD0512AD1BC72FE55C47564F930D':'eci1',
                    '9C73E6E97A2CBF51E4B278ADA23CCAE1':'earfcn1',
                    '9DDE07A2C1CF0744AC62A8DB599676EA':'pci1',
                    '5A0FBA8193649C6FFF16D3935F71CEF1':'routeIndex1',
                    'F054F07668E31882DDF58BE65F3D0ADF':'plmn1',
                    'C4606CDA2A3746764DE1D472B7CE3D68':'transmissionPowerFirst1',
                    '7E4A19F0DE1031751EDADD06687489E4':'transmissionPower1',
                    //lte cell3 第三组数据
                    '2':'lteCellIndex2', 
                    '23E6565424A1C1FF45150D7F84993B05':'enable2',
                    '7C87FB8969510076099530294913AA78':'cellStatus2',
                    '0720AB24B8821CC8F957D7D575A81672':'rfStatus2',
                    '80F2FDC7C7DA8B27D3577116F1E22663':'band2',
                    '769E811FC533C0FA00B3B3A3680A507D':'bandwidth2',
                    'C0C55B8AD23FCE6B55BD25428E3B06FD':'eci2',
                    '5A6465449F4B3A23FA73D39E53F66448':'earfcn2',
                    '76093ED979DC756A4C6A0D13F6064B5B':'pci2',
                    'B989518CC02E0C8B177EA7DFDF033FEC':'routeIndex2',
                    'A2ED4F3CDA4F67B0EDBC207FFC22164E':'plmn2',
                    '902678F40B8D9363C0CFE5783873EBAF':'transmissionPowerFirst2',
                    '8CB71F3A73F90E66C40DDA5EB72E51A7':'transmissionPower2',

                    //GSM Cell 1 第一组数据
                    'B73F47DE67E37BF2B5BE63AA1C207750': 'enable3',
                    '0': 'gsmCellIndex3',
                    '24D0A7AEFEAA50C3314B52574C6C3490': 'cellStatus3',//回显字段
                    'E1AC6968358B5D30BF68B91E4575A68C': 'cellStatusSet3',//下发字段
                    'E8C1977D2390D00E314FCBCEEA1154C5': 'rfStatus3',
                    'FD2FCBEF20CCD7A4C8E4A472E7B56339': 'abisLinkStatus3', 
                    'F46A81F4CCFB5D5A6DB7E15523809289': 'ueConnections3',
                    'E0AED6A11D7D84D99EC6C860ECDF623C': 'gsmCellID3',
                    '2597249B2AE412842E7DA4BE16B6DD7D': 'lac3', 
                    '197EDD9D84FCBA42331E5393899D5AA8': 'plmn3',
                    'F439E894612403F0E23A3DC834545E19': 'plmnSecond3',
                    '04B38E113761893AAC18DF5CF9840A4E': 'gsmBSIC3',
                    'B3F618C745142310AA63642047AC604A': 'routeIndex3',
                    '4718ADEC4F03961AF560F12591883D28': 'gsmArfcn3',
                    '7F571205DC8C7C659259C79B12FCC1DA': 'transmissionPower3',
                    '4E425A1133E632E477435DBDB3B500DC': 'ipaAndUnitId3', 
                    'A337CFB51408E6BD65350EC124E2BC16': 'remoteIpBackup3', 
                    '035333CD10D4B4ED8199CAAC7BF985AE': 'remoteIp3',
                    'FD0B70D0AB3A19922E8BCB15E962051A': 'cellDt3',
                    'B359A3D2EF75828AC107E690AA0C3409': 'traceMessageType3',
                    'E8EECDB3BA54DEC9A13E8006743BDE38': 'stubValue3',
                    'A757E69BA63C64D2F1CD977BFB1DD39F': 'traceNum3',
                    //gsm cell 2 第二组数据
                    '4B5CAA00DB794FB0F1698B07B515FDFC': 'enable4', 
                    '1': 'gsmCellIndex4',
                    '12EE74C011FE465A0767AF647616B46E': 'cellStatus4',//回显字段
                    '21283735716A56C3BB3523B189BDB5A7': 'cellStatusSet4',//下发字段
                    'B535A51BC4F9D433751EA841AE3486D3': 'rfStatus4',
                    'A65FD409CA9B4B060A21ED235E1E6947': 'abisLinkStatus4',
                    'C57D63E1295D803EF5CC8BCFA81B7176': 'ueConnections4',
                    'C845E16393E6B3A6BA3FBE55D96D366D': 'gsmCellID4',
                    '73F1062AF3D09D7F2C5357F6375F1638': 'lac4',
                    '189DFC545B7E119FCEF62885A0CC6E76': 'plmn4',
                    'E935EE58C870F2B2A07AC237F127A213': 'plmnSecond4',
                    'CC0B1F629047A0A8ACFAB54F0E01AF18': 'gsmBSIC4',
                    'BB01BEF153C1E795A3256D3A0824776A': 'routeIndex4',
                    'D6E04A8C3CA9DF1A76F12B164F9DC2E8': 'gsmArfcn4',
                    '35E2C50327ED27AB311EC16F0BC43781': 'transmissionPower4',
                    'E6F00CA17B00105DB30C83BCF372D782': 'ipaAndUnitId4', 
                    '80949A36A5A392D70F47DD225AE9F43B': 'remoteIpBackup4', 
                    '0D54899E2485751F02D75A215D691127': 'remoteIp4',
                    '99ADB03362E84CC9F73CD5E75E3D5FBA': 'cellDt4',
                    '1DEBC76AD179724CEA635A76CB8334BD': 'traceMessageType4',
                    'E67FB531C55DBD25FB7617ABF2B8FDA3': 'stubValue4',
                    '1D1366B9639FB8D112D9E3F64D1D762C': 'traceNum4',
                    //GSM Cell 3 第三组数据  
                    '335947E4686BF45EA915EB1E8BA26A5C': 'enable5', 
                    '2': 'gsmCellIndex5',
                    '649DB6EDAB7D6BA9C843E3551D14EFEE': 'cellStatus5', //回显字段
                    '91CEFAB01C50F8F923527F71264BC321': 'cellStatusSet5',//下发字段
                    'F7621DAF8D639F5702F078660ACCE6AF': 'rfStatus5',  
                    '4DAD2EE1D2488B9A9EA539EDFE22FB03': 'abisLinkStatus5', 
                    '2DE1B299F831D2E9241DD86F03D0887C': 'ueConnections5',
                    '86CD5D619D0DE75FD055AF2BC17EA207': 'gsmCellID5',
                    'AF9DEC144EC0B26E2AB0B8EC8A85F5FD': 'lac5',
                    '7A6178E2D57852AF3FFF3794D5330A31': 'plmn5',
                    '36308BB33D04CBFC6668D967C916808A': 'plmnSecond5',
                    '721A9B84381B03096D588E21315A9785': 'gsmBSIC5', 
                    '1181E9B7483BE5EB820F6385EEF57305': 'routeIndex5', 
                    '991C7D9E8CC239E9395F0FBC3DF46EE0': 'gsmArfcn5',
                    'DECA56273FD10BC5279CD75EAB2803C8': 'transmissionPower5', 
                    'A25518A568455BBB06344D38A61668F1': 'ipaAndUnitId5', 
                    'F93888065D46D9DF52BB78FECA746721': 'remoteIpBackup5',  
                    '8CCA4FF9F657D051320E12C3DF33C437': 'remoteIp5',
                    'B91CAE826F2213929C6C763D31B6E200': 'cellDt5',
                    '238FCBE56757C2AC26C99723C0BAD267': 'traceMessageType5',
                    '679902758EC5E0F3038049CF89E037B2': 'stubValue5',
                    'A3359824CFD213BCFF34388F778D1D46': 'traceNum5',
				},
                // Band-Bandwidth-EARFCN 映射关系数据
                bandBandwidthEarfcnMapping: {
                    '3': {
                        name: '3',
                        frequencyRange: { dl: [1805, 1880], ul: [1710, 1785] },
                        bandwidths: {
                            '25': { label: '5MHz', earfcnRange: [1200, 1950] },
                            '50': { label: '10MHz', earfcnRange: [1200, 1950] },
                            '75': { label: '15MHz', earfcnRange: [1200, 1950] },
                            '100': { label: '20MHz', earfcnRange: [1200, 1950] }
                        }
                    },
                    '8': {
                        name: '8',
                        frequencyRange: { dl: [925, 960], ul: [880, 915] },
                        bandwidths: {
                            '25': { label: '5MHz', earfcnRange: [3450, 3800] },
                            '50': { label: '10MHz', earfcnRange: [3450, 3800] },
                            '75': { label: '15MHz', earfcnRange: [3450, 3800] },
                            '100': { label: '20MHz', earfcnRange: [3450, 3800] }
                        }
                    },
                    '20': {
                        name: '20',
                        frequencyRange: { dl: [791, 821], ul: [832, 862] },
                        bandwidths: {
                            '25': { label: '5MHz', earfcnRange: [6150, 6450] },
                            '50': { label: '10MHz', earfcnRange: [6150, 6450] },
                            '75': { label: '15MHz', earfcnRange: [6150, 6450] },
                            '100': { label: '20MHz', earfcnRange: [6150, 6450] }
                        }
                    },
                    '28': {
                        name: '28',
                        frequencyRange: { dl: [758, 803], ul: [703, 748] },
                        bandwidths: {
                            '25': { label: '5MHz', earfcnRange: [9210, 9660] },
                            '50': { label: '10MHz', earfcnRange: [9210, 9660] },
                            '75': { label: '15MHz', earfcnRange: [9210, 9660] },
                            '100': { label: '20MHz', earfcnRange: [9210, 9660] }
                        }
                    }
                },
            }
		},
        computed: {
            //根据 casts 生成表格
            lteCellTbList: function(){
                var vm = this,
                    list = [],
                    row = vm.lteRowData;

                list = [{ //lte cell 1 第一组数据
                    enable: row['17C1B28826341D21A9695446B54528D5'], 
                    lteCellIndex: '0',
                    cellStatus: row['53550CDE5EFC208C01C6DB23CF99A242'],
                    rfStatus: row['C16F053DAD76488940AE36DAFAB05DD8'],
                    band: row['19E793A240D2E04EFB520BA1415C6C8E'],
                    bandwidth: row['3B9BCB20DA630BB909F37807CC3A2CA7'],
                    eci: row['B471C10C1E3882E9E404B5D8E4E5C79F'],
                    earfcn: row['6BFB6D72A2C84375FEC722F36E843A50'],
                    transmissionPowerFirst: row['C488255FC869387264FAB93612C70B49'],
                    transmissionPower: row['AE2D44FB529D4CCBD6B198D5274355D0'],
                    pci: row['52B7BDBB5D1B07AFBEAA958A0BE841B9'],
                    routeIndex: row['702F9993EBFF3B9544220ABF24349D71'],
                    plmn: row['CE6801DEF400E2A1BBFD60CBD23B5380'],
                    
                },{//lte cell 2 第二组数据
                    enable: row['97442F3B1E08A9B0616C9813EC17E45C'], 
                    lteCellIndex: '1',
                    cellStatus: row['2B46400422F9D5E7F94EEE9B7F373607'],
                    rfStatus: row['D22497BF63292C42A05EBB10039AD568'],
                    band: row['4D601E9CE78DE9173728E2E2A100371E'],
                    bandwidth: row['B7603FA1E0908A6C9B93395C80B27F98'],
                    eci: row['7593BD0512AD1BC72FE55C47564F930D'],
                    earfcn: row['9C73E6E97A2CBF51E4B278ADA23CCAE1'],
                    transmissionPowerFirst: row['C4606CDA2A3746764DE1D472B7CE3D68'],
                    transmissionPower: row['7E4A19F0DE1031751EDADD06687489E4'],
                    pci: row['9DDE07A2C1CF0744AC62A8DB599676EA'],
                    routeIndex: row['5A0FBA8193649C6FFF16D3935F71CEF1'],
                    plmn: row['F054F07668E31882DDF58BE65F3D0ADF'],
                    
                },{//lte cell 3 第三组数据
                    enable: row['23E6565424A1C1FF45150D7F84993B05'], 
                    lteCellIndex: '2',
                    cellStatus: row['7C87FB8969510076099530294913AA78'],
                    rfStatus: row['0720AB24B8821CC8F957D7D575A81672'],
                    band: row['80F2FDC7C7DA8B27D3577116F1E22663'],
                    bandwidth: row['769E811FC533C0FA00B3B3A3680A507D'],
                    eci: row['C0C55B8AD23FCE6B55BD25428E3B06FD'],
                    earfcn: row['5A6465449F4B3A23FA73D39E53F66448'],
                    transmissionPowerFirst: row['902678F40B8D9363C0CFE5783873EBAF'],
                    transmissionPower: row['8CB71F3A73F90E66C40DDA5EB72E51A7'],
                    pci: row['76093ED979DC756A4C6A0D13F6064B5B'],
                    routeIndex: row['B989518CC02E0C8B177EA7DFDF033FEC'],
                    plmn: row['A2ED4F3CDA4F67B0EDBC207FFC22164E'],
                    
                }];
                
                // 只展示 enable 为 '1' 的数据
                return list.filter(item => item.enable == '1');
            }, 
            gsmCellTbList: function(){
                var vm = this,
                    list = [],
                    row = vm.gsmRowData; //gsmRowData 中读取回显字段的值来构建表格数据：

                list = [
                    {    //gsm cell 1
                        enable: row['B73F47DE67E37BF2B5BE63AA1C207750'], 
                        gsmCellIndex: '0',
                        cellStatus: row['24D0A7AEFEAA50C3314B52574C6C3490'],
                        rfStatus: row['E8C1977D2390D00E314FCBCEEA1154C5'],
                        abisLinkStatus: row['FD2FCBEF20CCD7A4C8E4A472E7B56339'],
                        ueConnections: row['F46A81F4CCFB5D5A6DB7E15523809289'],
                        gsmCellID: row['E0AED6A11D7D84D99EC6C860ECDF623C'],
                        lac: row['2597249B2AE412842E7DA4BE16B6DD7D'],
                        plmn: row['197EDD9D84FCBA42331E5393899D5AA8'],
                        plmnSecond: row['F439E894612403F0E23A3DC834545E19'],
                        gsmBSIC: row['04B38E113761893AAC18DF5CF9840A4E'],
                        routeIndex: row['B3F618C745142310AA63642047AC604A'],
                        gsmArfcn: row['4718ADEC4F03961AF560F12591883D28'],
                        transmissionPower: row['7F571205DC8C7C659259C79B12FCC1DA'],
                        ipaAndUnitId: row['4E425A1133E632E477435DBDB3B500DC'],
                        remoteIpBackup: row['A337CFB51408E6BD65350EC124E2BC16'],
                        remoteIp: row['035333CD10D4B4ED8199CAAC7BF985AE'],
                        cellDt: row['FD0B70D0AB3A19922E8BCB15E962051A'],
                        traceMessageType: row['B359A3D2EF75828AC107E690AA0C3409'],
                        stubValue: row['E8EECDB3BA54DEC9A13E8006743BDE38'],
                        traceNum: row['A757E69BA63C64D2F1CD977BFB1DD39F'],
                    },
                    {    //gsm cell 2
                        enable: row['4B5CAA00DB794FB0F1698B07B515FDFC'],
                        gsmCellIndex: '1',
                        cellStatus: row['12EE74C011FE465A0767AF647616B46E'],
                        rfStatus: row['B535A51BC4F9D433751EA841AE3486D3'],
                        abisLinkStatus: row['A65FD409CA9B4B060A21ED235E1E6947'],
                        ueConnections: row['C57D63E1295D803EF5CC8BCFA81B7176'],
                        gsmCellID: row['C845E16393E6B3A6BA3FBE55D96D366D'],
                        lac: row['73F1062AF3D09D7F2C5357F6375F1638'],
                        plmn: row['189DFC545B7E119FCEF62885A0CC6E76'],
                        plmnSecond: row['E935EE58C870F2B2A07AC237F127A213'],
                        gsmBSIC: row['CC0B1F629047A0A8ACFAB54F0E01AF18'],
                        routeIndex: row['BB01BEF153C1E795A3256D3A0824776A'],
                        gsmArfcn: row['D6E04A8C3CA9DF1A76F12B164F9DC2E8'],
                        transmissionPower: row['35E2C50327ED27AB311EC16F0BC43781'],
                        ipaAndUnitId: row['E6F00CA17B00105DB30C83BCF372D782'],
                        remoteIpBackup: row['80949A36A5A392D70F47DD225AE9F43B'],
                        remoteIp: row['0D54899E2485751F02D75A215D691127'],
                        cellDt: row['99ADB03362E84CC9F73CD5E75E3D5FBA'],
                        traceMessageType: row['1DEBC76AD179724CEA635A76CB8334BD'],
                        stubValue: row['E67FB531C55DBD25FB7617ABF2B8FDA3'],
                        traceNum: row['1D1366B9639FB8D112D9E3F64D1D762C'],
                    },
                    {   //gsm cell 3
                        enable: row['335947E4686BF45EA915EB1E8BA26A5C'],
                        gsmCellIndex: '2',
                        cellStatus: row['649DB6EDAB7D6BA9C843E3551D14EFEE'],
                        rfStatus: row['F7621DAF8D639F5702F078660ACCE6AF'],
                        abisLinkStatus: row['4DAD2EE1D2488B9A9EA539EDFE22FB03'],
                        ueConnections: row['2DE1B299F831D2E9241DD86F03D0887C'],
                        gsmCellID: row['86CD5D619D0DE75FD055AF2BC17EA207'],
                        lac: row['AF9DEC144EC0B26E2AB0B8EC8A85F5FD'],
                        plmn: row['7A6178E2D57852AF3FFF3794D5330A31'],
                        plmnSecond: row['36308BB33D04CBFC6668D967C916808A'],
                        gsmBSIC: row['721A9B84381B03096D588E21315A9785'],
                        routeIndex: row['1181E9B7483BE5EB820F6385EEF57305'],
                        gsmArfcn: row['991C7D9E8CC239E9395F0FBC3DF46EE0'],
                        transmissionPower: row['DECA56273FD10BC5279CD75EAB2803C8'],
                        ipaAndUnitId: row['A25518A568455BBB06344D38A61668F1'],
                        remoteIpBackup: row['F93888065D46D9DF52BB78FECA746721'],
                        remoteIp: row['8CCA4FF9F657D051320E12C3DF33C437'],
                        cellDt: row['B91CAE826F2213929C6C763D31B6E200'],
                        traceMessageType: row['238FCBE56757C2AC26C99723C0BAD267'],
                        stubValue: row['679902758EC5E0F3038049CF89E037B2'],
                        traceNum: row['A3359824CFD213BCFF34388F778D1D46'],
                    }
                ]
                
                // 只展示 enable 为 '1' 的数据
                return list.filter(item => item.enable == '1');
            },
            //cell config option
            cellOptions() {
                var vm = this,
                    list = [],
                    lteRowData = vm.lteRowData,
                    gsmRowData = vm.gsmRowData;
                
                // lte cell options
                list = [{ 
                    text: 'LTE Cell 0', 
                    value: 'LTE Cell 0',
                    enable: lteRowData['17C1B28826341D21A9695446B54528D5'],
                    index: 0
                },{ 
                    text: 'LTE Cell 1', 
                    value: 'LTE Cell 1',
                    enable: lteRowData['97442F3B1E08A9B0616C9813EC17E45C'],
                    index: 1
                },{
                    text: 'LTE Cell 2', 
                    value: 'LTE Cell 2',
                    enable: lteRowData['23E6565424A1C1FF45150D7F84993B05'],
                    index: 2
                },
                // gsm cell options
                { 
                    text: 'GSM Cell 0', 
                    value: 'GSM Cell 0',
                    enable: gsmRowData['B73F47DE67E37BF2B5BE63AA1C207750'],
                    index: 3
                },{ 
                    text: 'GSM Cell 1', 
                    value: 'GSM Cell 1',
                    enable: gsmRowData['4B5CAA00DB794FB0F1698B07B515FDFC'],
                    index: 4
                },{
                    text: 'GSM Cell 2', 
                    value: 'GSM Cell 2',
                    enable: gsmRowData['335947E4686BF45EA915EB1E8BA26A5C'],
                    index: 5
                }];

                return list;
            },
            //动态生成 ipaAndUnitId 字段
            ipaAndUintId() {
                var vm = this;

                return vm.ipaValue + '-' + vm.unitIdValue;
            },
            
            
            // 获取当前Band和Bandwidth下的EARFCN范围（动态计算）
            currentEarfcnRange() {
                var vm = this;

                if (!vm.editLteCellForm.band || !vm.editLteCellForm.bandwidth || 
                    !vm.bandBandwidthEarfcnMapping[vm.editLteCellForm.band]) {
                    return null;
                }
                var bandConfig = vm.bandBandwidthEarfcnMapping[vm.editLteCellForm.band];
                if (!bandConfig.bandwidths[vm.editLteCellForm.bandwidth]) {
                    return null;
                }
                
                // 获取基础的 EARFCN 范围
                var baseRange = bandConfig.bandwidths[vm.editLteCellForm.bandwidth].earfcnRange,
                    bandwidth = parseInt(vm.editLteCellForm.bandwidth), // 25, 50, 75, 100
                    minEarfcn = baseRange[0] + bandwidth,
                    maxEarfcn = baseRange[1] - bandwidth;
                
                return [minEarfcn, maxEarfcn];
            },
        },
		watch: {
			'editLteCellForm.earfcn': function(newValue){
                var vm = this,
                    match = newValue.match(/\(([^)]+)\)/);

                // 先判断是否携带小括号
                if (newValue.indexOf('(') !== -1 && newValue.indexOf(')') !== -1) {
                    var match = newValue.match(/\(([^)]+)\)/);

                    vm.lteFrequencyValue = match ? match[1] : '';
                } else {
                    vm.lteFrequencyValue = vm.formatConversion(newValue, vm.editLteCellForm.band);
                }
            },
            ipaAndUintId: function(val) {
                var vm = this;

                vm.editGsmCellForm.ipaAndUnitId = val;
            },
		},
		methods:{ 
            // 初始化          
			init(code,id){
				var vm = this,
                    powerLists = ['36dBm','37dBm','38dBm','39dBm','40dBm','41dBm','42dBm','43dBm','44dBm','45dBm','46dBm'],
                	powerValLists = [36,37,38,39,40,41,42,43,44,45,46];
				
				vm.txfPowerList = powerLists.map((item,index)=>{
					return { label: item, value: powerValLists[index] }
				})

				vm.smallCellCode = code;
				vm.getParamNode(code,id);
			},
            
            // 获取参数节点及数据
			getParamNode(code,id) {
                var vm = this,
                    codes = [],
                    url = '${ctx}/cell/quicksettings/getParamNodeTreeAndData.action',
                    params = {
                        id: id,
                        smallCellCode: code
                    };

                axios.post(url, stringify(params)).then(function(res){
                    var data = res.data;
                    
                    if(data && Array.isArray(data)) {
                        data.map(function(item){
                            // 根据 title 取 lte cell 值
                            if(item.title == 'LTE Cell Settings'){
                               //根据接口返回的 groups 
                                vm.lteRowData = {};
                                item.groups.map(function(group){
                                    group.list.map(function(m){
                                        //收集所有的 code 方便后续提交使用
                                        codes.push(m.name);

                                        //这里需要重新组织数据结正确的显示表格数据 lte Cell tableList
                                        vm.$set(vm.lteRowData, m.name, m.value);
                                        
                                    })
                                })
                                Object.assign(vm.originLteRowData, vm.lteRowData);
                            }
                            //gsm cell table data 
                            if(item.title == 'GSM Cell Settings'){
                                vm.gsmRowData = {};
                                item.groups.map(function(group){
                                    group.list.map(function(m){
                                        //收集所有的 code 方便后续提交使用
                                        codes.push(m.name);

                                        //这里需要重新组织数据结正确的显示表格数据 gsm  Cell tableList
                                        vm.$set(vm.gsmRowData, m.name, m.value);
                                    })
                                })
                                Object.assign(vm.originGsmRowData, vm.gsmRowData);
                            }
                        });

						vm.$nextTick(function(){
                            initForm(vm.$refs.cellConfigForm);
							$('#setting_main').removeClass('loading');
                        });

                        vm.codeList = codes;
                    }
                });
            },
            
            // 根据属性名获取参数名
            getNameByProp(prop) {
                var vm = this,
                    reg = /^\w*\.\d*\.\w*$/,
                    key = prop;
                
                if(reg.test(prop)) {
                    var mReg = /\.(\d*)\./,
                    
                        sufReg = /\.(\w*)$/,
                        idx = prop.match(mReg)[1],
                        sufStr = prop.match(sufReg)[1];

                    vm.codeList.map(function(name){
                        var index = vm.indexs[name];
                        if(vm.casts[name] == sufStr && index == idx) {
                            key = name;
                        }
                    });
                }else {
                    (vm.codeList.concat(vm.codeTableList)).map(function(name){
                        if(vm.casts[name] == prop) {
                            key = name;
                        }
                    });
                }

                return key;
            },
            //带宽回显
            bandwidthFmtOne(row,column,value,index) {
				var vm = this,
					rels = {
						'25': '5MHz',
						'50': '10MHz',
						'75': '15MHz',
						'100': '20MHz'
					};
				return rels[value]||'';
			},
            // 添加小区配置
            addCellConfig(){
                var vm = this;

                if(vm.cellConfigAdd !== '' && vm.cellConfigAdd !== null && vm.cellConfigAdd !== undefined){
                    vm.cellConfigShow = true;
                } else {
                   vm.cellConfigShow = false;
                }
			},
            // 提交添加小区配置
            addCellConfigSubmit(){
                var vm = this,
                    index = vm.cellConfigAdd + '',
                    groupMap = {
                        0: '17C1B28826341D21A9695446B54528D5',
                        1: '97442F3B1E08A9B0616C9813EC17E45C',
                        2: '23E6565424A1C1FF45150D7F84993B05',

                        3: 'B73F47DE67E37BF2B5BE63AA1C207750',
                        4: '4B5CAA00DB794FB0F1698B07B515FDFC',
                        5: '335947E4686BF45EA915EB1E8BA26A5C'
                    };
                
                //配置成功后，将 cell config 下拉的数据显示为置灰状态-不可选；
                if(['0','1','2'].includes(index)) vm.lteRowData[groupMap[index]] = '1';
                
                if(['3','4','5'].includes(index)) vm.gsmRowData[groupMap[index]] = '1';

                // 只有在成功添加后才清空选择
                vm.cellConfigAdd = '';
                vm.closeCellConfigDialog();
            },
            // 关闭添加小区配置对话框
			closeCellConfigDialog() {
                var vm = this;

                vm.cellConfigShow = false;
            },
            // 修改 lte 小区
            editLteCellClick(row){
                var vm = this;

                //特殊处理 plmn 字段回显
                vm.defaultPlmnList = row.plmn ? row.plmn.split(',').map(function(item){
                    return item;
                }) : [];
                //回显 cell id： 根据 eci 进行计算
                if(row.eci !== undefined){
                    var eci = row.eci*1,
                        cellId = eci % 256;
                    vm.lteCellIdValue = cellId;
                }
                //回显频率，根据频点和band进行转换，无需接口返回值
                if(row.earfcn && row.band){
                    vm.lteFrequencyValue = vm.formatConversion(row.earfcn, row.band);
                }

                Object.assign(vm.editLteCellForm, row)
                
                // 确保 transmissionPower 为数字类型，以便与下拉选项的 value 正确匹配
                if (vm.editLteCellForm.transmissionPower !== undefined && vm.editLteCellForm.transmissionPower !== null) {
                    vm.editLteCellForm.transmissionPower = parseInt(vm.editLteCellForm.transmissionPower);
                }
                
                vm.editLteCellShow = true;
			},
            // 提交修改 lte 小区
            editLteCellSubmit() {
                var vm = this;

                vm.$refs.editLteCellForm.validate(function(valid){
                    if(valid){
                        // 自动获取 editLteCellForm 的所有 key，排除空值 
                        var row = {},
                            codes = Object.keys(vm.editLteCellForm).filter(function(key){
                                return vm.editLteCellForm[key] !== undefined;
                            });

                        codes.map(function(code){
                            if(!code.includes('lteCellIndex')) {
                                var name = vm.getNameByProp(code + vm.editLteCellForm['lteCellIndex']);
                                row[name] = vm.editLteCellForm[code];
                            }
                        })        

                        Object.assign(vm.lteRowData, row)
                        vm.editLteCellClose();
                    }
                })
			
            },
			// 关闭修改 lte 小区对话框
            editLteCellClose() {
                var vm = this;

                vm.$refs.editLteCellForm.clearValidate();
                
                vm.editLteCellShow = false;
            },
			delLteCellClick(row){
				var vm = this;
				var confirmStr = '<%=rb.getString("QueRenShanChu")%>'
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(()=>{
                    //删除某条数据，将对应的开关打开，cell config 中可进行选择
                    var index = row.lteCellIndex + '',
                        groupMap = {
                            0: '17C1B28826341D21A9695446B54528D5',
                            1: '97442F3B1E08A9B0616C9813EC17E45C',
                            2: '23E6565424A1C1FF45150D7F84993B05'
                        };
                    
                    vm.lteRowData[groupMap[index]] = '0';
				})
			},
            
           // Band 选择变化处理
            onBandChange() {
                var vm = this;

                vm.updateEarfcnValidation();
            },
            // Bandwidth 选择变化处理  
            onBandwidthChange() {
                var vm = this;

                vm.updateEarfcnValidation();
            },
            // 更新 EARFCN 验证规则
            updateEarfcnValidation() {
                var vm = this,
                    range = vm.currentEarfcnRange; 
                
                //页面提示的频点范围,通过表单校验该字段
                if (range) {
                    vm.$refs.editLteCellForm.validateField('earfcn');
                }
            },
            
            // EARFCN 输入事件处理
            onEarfcnInput(value) {
                var vm = this;

                if (value && vm.editLteCellForm.band) {
                    vm.lteFrequencyValue = vm.formatConversion(value, vm.editLteCellForm.band);
                } else {
                    vm.lteFrequencyValue = '';
                }
            },
            // 根据 Band 和 EARFCN 计算频率 格式转换
            formatConversion(earfcn, band) {
                var vm = this;
                
                if (!band || !earfcn) return '';
                
                var earfcnNum = parseInt(earfcn),
                    bandConfig = vm.bandBandwidthEarfcnMapping[band];

                if (!bandConfig) return earfcn + '';
                
                // 根据不同的 Band 计算频率
                switch (band) {
                    case '3': // Band 3 (1800 MHz)
                        if (earfcnNum >= 1200 && earfcnNum <= 1950) {
                            var frequency = 1805 + 0.1 * (earfcnNum - 1200);
                            return vm.formatFrequency(frequency);
                        }
                        break;
                    case '8': // Band 8 (900 MHz)  
                        if (earfcnNum >= 3450 && earfcnNum <= 3800) {
                            var frequency = 925 + 0.1 * (earfcnNum - 3450);
                            return vm.formatFrequency(frequency);
                        }
                        break;
                    case '20': // Band 20 (800 MHz)
                        if (earfcnNum >= 6150 && earfcnNum <= 6450) {
                            var frequency = 791 + 0.1 * (earfcnNum - 6150);
                            return vm.formatFrequency(frequency);
                        }
                        break;
                    case '28': // Band 28 (700 MHz)
                        if (earfcnNum >= 9210 && earfcnNum <= 9660) {
                            var frequency = 758 + 0.1 * (earfcnNum - 9210);
                            return vm.formatFrequency(frequency);
                        }
                        break;
                    default:
                        return earfcn + '';
                }
                return earfcn + '';
            },

            // 智能格式化频率的辅助函数
            formatFrequency(frequency) {
                // 整数时不显示小数点，有小数时显示小数
                return (frequency % 1 === 0 ? frequency.toFixed(0) : frequency.toFixed(1));
            },
            // 修改 gsm 小区
			editGsmCellClick(row){
                var vm = this;

                //回显频率
                if(row.gsmArfcn){
                    if(row.gsmArfcn !== undefined && row.gsmArfcn !== null && row.gsmArfcn !== "") {
                        var gsmArfcn = parseInt(row.gsmArfcn, 10);
                        var uplink = null, downlink = null;
                        if(gsmArfcn >= 0 && gsmArfcn <= 124){
                            uplink = 890.0 + 0.2 * gsmArfcn;
                            downlink = 935.0 + 0.2 * gsmArfcn;
                        }else if(gsmArfcn >= 975 && gsmArfcn <= 1023){
                            uplink = 890.0 + 0.2 * (gsmArfcn - 1024);
                            downlink = 935.0 + 0.2 * (gsmArfcn - 1024);
                        }
                        if(uplink !== null && downlink !== null){
                            vm.gsmFrequencyValue = 'Uplink: ' + uplink + " / " + "Downlink: " + downlink;
                        }else{
                            vm.gsmFrequencyValue = "";
                        }
                    }else{
                        vm.gsmFrequencyValue = "";
                    }
                }
                //ipa+unit id
                if(row.ipaAndUnitId !== undefined && row.ipaAndUnitId !== null && row.ipaAndUnitId !== ""){
                    var ipaStr = String(row.ipaAndUnitId);
                    var arr = ipaStr.split('-');
                    
                    vm.ipaValue = arr[0] ? arr[0] : '';
                    vm.unitIdValue = arr[1] ? arr[1] : '';
                }
                
				Object.assign(vm.editGsmCellForm, row)
                
                // 确保 transmissionPower 为数字类型，以便与下拉选项的 value 正确匹配
                if (vm.editGsmCellForm.transmissionPower !== undefined && vm.editGsmCellForm.transmissionPower !== null) {
                    vm.editGsmCellForm.transmissionPower = parseInt(vm.editGsmCellForm.transmissionPower);
                }
                
                vm.editGsmCellShow = true;
			},
            // 添加事件 Default plmn
            addDefaultPlmn(){
                var vm = this,
                    val = vm.defaultPlmn.plmn,
                    reg = /^[0-9]{5,6}$/;

                if(val){
                    if(reg.test(val)) {
                        var result = vm.defaultPlmnList.some(item=>item == val);
                        if(result){
                            vm.defaultPlmn.defaultPlmnErrorMessage = '<%=rb.getString("YiCunZai")%>';
                        }else{
                            vm.defaultPlmnList.push(val);
                            vm.defaultPlmn.plmn = '';
                            vm.defaultPlmn.defaultPlmnErrorMessage = '';
                            vm.editLteCellForm.plmn = vm.defaultPlmnList.join(',');
                        }
                    }else {
                        vm.defaultPlmn.defaultPlmnErrorMessage = 'Length<%=rb.getString("MaoHao")%> 5~6 Digit <%=rb.getString("ZhengXing")%>';
                    }
                }
                
            },
            // Default Route plmn 删除事件
            defaultPlmnDel(val){
                var vm = this;
                vm.defaultPlmnList = vm.defaultPlmnList.filter((items)=>{
                    return items != val
                })
                vm.editLteCellForm.plmn = vm.defaultPlmnList.join(',');
            },
            // 提交修改 gsm 小区
            editGsmCellSubmit() {
                var vm = this;
                // 先校验 ipaValue 和 unitIdValue
                if(vm.unitIdError || vm.ipaValueError){
                    return;
                }
                vm.$refs.editGsmCellForm.validate(function(valid) {
                    if(valid){
                        var row = { },
                            codes = Object.keys(vm.editGsmCellForm).filter(function(key){
                                return vm.editGsmCellForm[key] !== undefined;
                            });
						    
						codes.map(function(code){ 
                            if(!code.includes('gsmCellIndex')) {
                                var name = vm.getNameByProp(code + (vm.editGsmCellForm['gsmCellIndex']*1 + 3));
                                row[name] = vm.editGsmCellForm[code];
                            }
						})

						Object.assign(vm.gsmRowData, row);
                        vm.editGsmCellClose();
					}
                });
            },
            validateUnitId( min, max){
                var vm = this,
                    value = vm.unitIdValue,
                    reg = /^(0|[1-9][0-9]*)$/; 

                if(value == '' || !reg.test(value) || value < min || value > max){
                    vm.unitIdError = true;
                    vm.unitIdOk = false; 
                }else{
                    vm.unitIdError = false;
                    vm.unitIdOk = true;
                }
            },
            validateIpa(min, max){
                var vm = this,
                    value = vm.ipaValue,
                    reg = /^(0|[1-9][0-9]*)$/;

                if(value == '' || !reg.test(value) || value < min || value > max){
                    vm.ipaValueError = true;
                    vm.ipaValueOk = false;
                }else{
                    vm.ipaValueError = false;
                    vm.ipaValueOk = true;
                }
            },
            editGsmCellClose() {
                var vm = this;

                vm.$refs.editGsmCellForm.clearValidate(); 
                vm.editGsmCellShow = false;
            },
			delGsmCell(row){
				var vm = this;
				var confirmStr = '<%=rb.getString("QueRenShanChu")%>'
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(()=>{
                    
                    var index = row.gsmCellIndex + '',
                        groupMap = {
                            0: 'B73F47DE67E37BF2B5BE63AA1C207750',
                            1: '4B5CAA00DB794FB0F1698B07B515FDFC',
                            2: '335947E4686BF45EA915EB1E8BA26A5C'
                        };
                    
                    vm.gsmRowData[groupMap[index]] = '0';
				})
			},
			isNull(val){
                if(val==undefined || val == null || val =="") return true;
                else return false;
            },
             //校验IP
            isValidIP(ip){
				var reg =  /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/     
				return reg.test(ip);     
			},
            checkCellPropsChange() {
                var vm = this,
                    isChanged = false;

                for(var key in vm.lteRowData) {
                    if(vm.lteRowData[key] != vm.originLteRowData[key]) {
                        isChanged = true;
                        break;
                    }
                }
                for(var key in vm.gsmRowData) {
                    if(vm.gsmRowData[key] != vm.originGsmRowData[key]) {
                        isChanged = true;
                        break;
                    }
                }

                return isChanged;
            },
            getCellChangedProps() {
                var vm = this,
                    params = {};

                for(var key in vm.lteRowData) {
                    if(vm.lteRowData[key] != vm.originLteRowData[key]) {
                        params[key] = vm.lteRowData[key];
                    }
                }
                for(var key in vm.gsmRowData) {
                    if(vm.gsmRowData[key] != vm.originGsmRowData[key]) {
                        var paramKey = key;
                        // 如果是 cellStatus 字段被修改，则转换为对应的下发字段ID
                        // GSM Cell 0: 24D0A7... → E1AC6968...
                        // GSM Cell 1: 12EE74... → 21283735...
                        // GSM Cell 2: 649DB6... → 91CEFAB0...
                        if(key === '24D0A7AEFEAA50C3314B52574C6C3490') {
                            paramKey = 'E1AC6968358B5D30BF68B91E4575A68C';
                        } else if(key === '12EE74C011FE465A0767AF647616B46E') {
                            paramKey = '21283735716A56C3BB3523B189BDB5A7';
                        } else if(key === '649DB6EDAB7D6BA9C843E3551D14EFEE') {
                            paramKey = '91CEFAB01C50F8F923527F71264BC321';
                        }
                        params[paramKey] = vm.gsmRowData[key];
                    }
                }

                return params;
            },
			save(){
                   var vm = this;
                   var params = {},
                       isCellChanged = vm.checkCellPropsChange(),
                       isChanged = isCellChanged;

                   if(!isChanged){
                       showMsg('prompt_msg','<%=rb.getString("CanShuZhiMeiYouBianHua")%>');
                       return;
                   }

                   // 提交小区变更参数
                   if(isCellChanged) {
                       var cellChangedProps = vm.getCellChangedProps();
                       Object.assign(params, cellChangedProps);
                   }

                   vm.$refs.cellConfigForm.validate(function(valid){
                       if(valid) {
                           var rowCode = vm.smallCellCode,
                               url = '${ctx}/cell/quicksettings/saveParamValue.action?smallCellCode='+rowCode;
                           $('#setting_main').addClass('loading');

                           axios.post(url,stringify({"params": JSON.stringify(params)})).then(res=>{
                               var data = res.data;
                               if(data["success"]){
                                   vm.$message.success({type:'success',message:'<%=rb.getString("ChengGong")%>'});
                                   closeSettingPanel();
                               }else{
                                   vm.$message.error(data["message"])
                               }
                               $('#setting_main').removeClass('loading');
                               settingVue.submitDisabled = false;
                           })
                       }
                   });
               },
			cancel(){
				var vm = this;
				if(isFormChanged(vm.$refs.cellConfigForm)){//返回true为改变
					vm.$confirm("<%=rb.getString("QueDingLiKaiDangQianYeMian")%>",'<%=rb.getString("QueRen")%>',{
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						cancelButtonText:'<%=rb.getString("QuXiao")%>',
						type:'warning',
						closeOnClickModal:false
					}).then(() => {
						closeSettingPanel();
					}).catch(() => {})
				}else{
					closeSettingPanel();
				}
			}
		},
		mounted(){
			eventBus.$off('tab-param').$on('tab-param',this.init);
			eventBus.$off('save-set').$on('save-set',this.save);
			eventBus.$off('cancel-set-tab').$on('cancel-set-tab',this.cancel);
		}
	})
</script>
