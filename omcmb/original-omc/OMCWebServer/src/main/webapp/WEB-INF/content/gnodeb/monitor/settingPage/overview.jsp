<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
.borderPage {
	border:1px solid #d5dcec;
	border-radius:10px;
	height:100%;
	background:#fff;
}
.limitForm {
	padding:30px;
}
.limitForm .el-form-item__label {
	line-height:initial;
}
.error-color {
	color:#f56c6c;
}
#enbLimitPage .el-form-item__error {
	top:unset;
	left:0;
}
#gnbOverviewPage .commonSize12Texts {
	font-size: 12px; 
	color: rgba(0, 0, 0, 0.8);
	padding-top: 5px;
}
</style>

<div id="gnbOverviewPage" style='display:flex;flex-direction:column;'>
	<div>
		<div class='commonContentBox'>
			<div class="commonText14"><%=rb.getString("XiaoQuXinXi")%></div>
			<div class='commonFlexWarp' style='padding: 16px 0 20px;'>
				<div class='commonItemBoxs'>
					<span class='commonTemplateText12'><%=rb.getString("PinDuan")%></span>
					<span class='commonSize12Texts'>{{rowDataInfo.Band}}</span>
				</div>
				<div class='commonItemBoxs'>
					<span class='commonTemplateText12'><%=rb.getString("NRXiaoQuID")%></span>
					<span class='commonSize12Texts'>{{rowDataInfo.nr_cell_id}}</span>
				</div>
				<div class='commonItemBoxs'>
					<span class='commonTemplateText12'>PCI</span>
					<span class='commonSize12Texts'>{{rowDataInfo.PHYCELLID}}</span>
				</div>
				<div class='commonItemBoxs'>
					<span class='commonTemplateText12'>TAC</span>
					<span class='commonSize12Texts'>{{rowDataInfo.tac}}</span>
				</div>
			</div>
			<div class='commonFlexWarp'>
				<div class='commonItemBoxs'>
					<span class='commonTemplateText12'><%=rb.getString("NRPinDianShangXian")%></span>
					<span class='commonSize12Texts'>{{rowDataInfo.EARFCNULINUSE}}</span>
				</div>
				<div class='commonItemBoxs'>
					<span class='commonTemplateText12'><%=rb.getString("NRPinDianXiaXian")%></span>
					<span class='commonSize12Texts'>{{rowDataInfo.EARFCNDLINUSE}}</span>
				</div>
				<div class='commonItemBoxs'>
					<span class='commonTemplateText12'><%=rb.getString("CPETxPower")%></span>
					<span class='commonSize12Texts'>{{rowDataInfo.tx_power}}</span>
				</div>
				<div class='commonItemBoxs'>
					<span class='commonTemplateText12'> </span>
					<span class='commonSize12Texts'> </span>
				</div>
			</div>
		</div>

		<div class='commonContentBox' style="margin-top: 10px;">
			<div class="commonText14"><%=rb.getString("SASSheBeiXinXi")%></div>
			<div class='commonFlexWarp' style='padding: 16px 0 20px;'>
				<div class='commonItemBoxs'>
					<span class='commonTemplateText12'><%=rb.getString("XiaoZhanBianMa")%></span>
					<span class='commonSize12Texts'>{{rowDataInfo.serial_number}}</span>
				</div>
				<div class='commonItemBoxs'>
					<span class='commonTemplateText12'><%=rb.getString("GnodebId")%></span>
					<span class='commonSize12Texts'>{{rowDataInfo.gNBId}}</span>
				</div>
				<div class='commonItemBoxs'>
					<span class='commonTemplateText12'><%=rb.getString("5GZhanDianMingCheng")%></span>
					<span class='commonSize12Texts'>{{rowDataInfo.host_name}}</span>
				</div>
				<div class='commonItemBoxs'>
					<span class='commonTemplateText12'><%=rb.getString("YingJianBanBen")%></span>
					<span class='commonSize12Texts'>{{rowDataInfo.firmware_version}}</span>
				</div>
			</div>
			
			<div class='commonFlexWarp' style="padding: 0 0 20px;">
				<div class='commonItemBoxs'>
					<span class='commonTemplateText12'><%=rb.getString("SoftwareVersion")%></span>
					<span class='commonSize12Texts'>{{rowDataInfo.software_version}}</span>
				</div>
				<div class='commonItemBoxs'>
					<span class='commonTemplateText12'><%=rb.getString("ChanPinLeiXingBiaoZhi")%></span>
					<span class='commonSize12Texts'>{{rowDataInfo.product}}</span>
				</div>
				<div class='commonItemBoxs'>
					<span class='commonTemplateText12'><%=rb.getString("ChanPinMingCheng")%></span>
					<span class='commonSize12Texts'>{{rowDataInfo.product_name}}</span>
				</div>
				<div class='commonItemBoxs'>
					<span class='commonTemplateText12'><%=rb.getString("SheBeiXingHaoMing")%></span>
					<span class='commonSize12Texts'>{{rowDataInfo.module_type}}</span>
				</div>
			</div>

			<div class='commonFlexWarp' style="padding: 0 0 20px;">
				<div class='commonItemBoxs'>
					<span class='commonTemplateText12'><%=rb.getString("DiYiCiLianJieShiJian")%></span>
					<span class='commonSize12Texts'>{{rowDataInfo.first_online_time}}</span>
				</div>
				<div class='commonItemBoxs'>
					<span class='commonTemplateText12'><%=rb.getString("ShangCiLianJieShiJian")%></span>
					<span class='commonSize12Texts'>{{rowDataInfo.LASTINFORMTIME}}</span>
				</div>
				<div class='commonItemBoxs'>
					<span class='commonTemplateText12'><%=rb.getString("YunXingShiJian")%></span>
					<span class='commonSize12Texts'>{{rowDataInfo.up_time}}</span>
				</div>
				<div class='commonItemBoxs'>
					<span class='commonTemplateText12'><%=rb.getString("SheBeiZu")%></span>
					<span class='commonSize12Texts'>{{rowDataInfo.group_name}}</span>
				</div>
			</div>
            <div class='commonFlexWarp' style="padding: 0 0 20px;">
				<div class='commonItemBoxs'>
					<span class='commonTemplateText12'>{{currentRemarkLabel}}</span>
					<span class='commonSize12Texts'>{{rowDataInfo.remark}}</span>
				</div>
			</div>
		</div>

		<div class='commonFlex' style='padding: 10px 0;'>
			<div style='width: calc(60% - 10px); margin-right: 10px'>
				<div class='commonContentBox'>
					<div class="commonText14"><%=rb.getString("ZhuangTai")%></div>
					<div class='commonFlexWarp' style='padding: 16px 0 0px;'>
						<div class='commonItemBoxs'>
							<span class='commonTemplateText12'><%=rb.getString("AdminZhuangTai")%></span>
							<!-- 1-Locked 2-Unlocked 3-ShuttingDown -->
							<span class="commonSize12Texts" v-if="rowDataInfo.adminState == '1'">Locked</span>
							<span class="commonSize12Texts" v-if="rowDataInfo.adminState == '2'">Unlocked</span>
							<span class="commonSize12Texts" v-if="rowDataInfo.adminState == '3'">ShuttingDown</span>
						</div>
						<div class='commonItemBoxs'>
							<span class='commonTemplateText12'><%=rb.getString("gNBZhuangTai")%></span>
							<!-- 1-激活 0-未激活 -->
							<!--
							<span class='commonSize12Texts activeStatusItem' v-if='rowDataInfo.op_state == 1'><%= rb.getString("JiHuo")%></span>
							<span class='commonSize12Texts failedText' v-else-if='rowDataInfo.op_state == 0'><%= rb.getString("QuJiHuo")%></span>
							-->
							<div v-if="rowDataInfo.op_state && rowDataInfo.op_state.length > 1" style="display: flex;">
								<span v-if="judgeActiveStatusFat(rowDataInfo.op_state) == '1'" style='margin-right: 5px;'><%= rb.getString("JiHuo")%></span>
								<span v-if="judgeActiveStatusFat(rowDataInfo.op_state) == '2'" class='offOrOnStatusCls' style='margin-right: 5px;'><%= rb.getString("JiHuo")%></span>
								<span v-if="judgeActiveStatusFat(rowDataInfo.op_state) == '3'" class='offStatusCls' style='margin-right: 5px;'><%= rb.getString("QuJiHuo")%></span>
								<el-popover title="<%= rb.getString("DuoXiaoQuZhuangTai")%>" popper-class="cellActivePopoverClass" trigger="click" width='300'>
									<span style="color:#4d84ff;" slot="reference">
										[ {{judgeActiveNumOrAllNumFat(rowDataInfo.op_state,'active')}}/{{judgeActiveNumOrAllNumFat(rowDataInfo.op_state,'all')}} ]
									</span>
									<div style="padding:10px;border-top:1px solid #E9E9E9;min-height:100px;display:flex;flex-wrap:wrap;">
										<div v-for="(item,index) in parseCellAndRfStatus(rowDataInfo.op_state)" style="height:40px;display:flex;align-items: center;margin-left:10px;">
											Cell {{index+1}}:
											<span v-if=" item == '1'" class="onStatusBoxCls" style='margin-left: 5px;'><%= rb.getString("JiHuo")%></span>
											<span v-if=" item == '0'" class="offStatusBoxCls" style='margin-left: 5px;'><%= rb.getString("QuJiHuo")%></span>
										</div>
									</div>
								</el-popover>
							</div>
							<div v-else v-html="cellStateFormatter(rowDataInfo.op_state, rowDataInfo)"></div>
						</div>
						<div class='commonItemBoxs'>
							<span class='commonTemplateText12'><%=rb.getString("HaloBKaiGuan")%></span>
							<!-- 1-enable 0-disable  -->
							<span v-if="rowDataInfo.halob_flag == '1'" style="display: flex;align-items: center;">
								HaloB<span class='el-icon el-icon-status-enable' style='margin-left:3px;font-size: 20px;'></span>
							</span>
							<span v-else-if="rowDataInfo.halob_flag == '0'" style="display: flex;align-items: center;">
								HaloB<span class='el-icon el-icon-status-disable' style='margin-left:3px;font-size: 20px;'></span>
							</span>
							<span v-else>--</span>
						</div>
					</div>
					<div class='commonFlexWarp' style='padding: 16px 0 0px;'>
						<div class='commonItemBoxs'>
							<span class='commonTemplateText12'><%=rb.getString("UEShu")%></span>
							<!-- 1-激活 0-未激活 -->
							<span v-html="ueCountsFormatter(rowDataInfo, rowDataInfo.ue_count)"></span>
						</div>
						<div class='commonItemBoxs'>
							<span class='commonTemplateText12'><%=rb.getString("TongBuZhuangTai")%></span>
							<span class='commonSize12Texts'>{{rowDataInfo.synStatus}}</span>
						</div>
						<div class='commonItemBoxs'>
							<span class='commonTemplateText12'><%=rb.getString("MultiPLMNZhuangTai")%></span>
							<!-- 1-Enable 0-Disable -->
							<span class='commonSize12Texts activeStatusItem' v-if='rowDataInfo.multiPlmnEnable == 1'><%= rb.getString("QiYong")%></span>
							<span class='commonSize12Texts failedText' v-else-if='rowDataInfo.multiPlmnEnable == 0'><%= rb.getString("JinYong")%></span>
						</div>
					</div>
					<div class='commonFlexWarp' style='padding: 16px 0 0px;'>
						<div class='commonItemBoxs'>
							<span class='commonTemplateText12'><%=rb.getString("ShePinKaiGuanZhuangTai")%></span>
							<!-- on-开启 off-关闭 -->
							<div v-if="rowDataInfo.rf_status && parseCellAndRfStatus(rowDataInfo.rf_status).length > 1"  style="display: flex;">
								<span v-if="judgeActiveStatusFat(rowDataInfo.rf_status) == '1'" style='margin-right: 5px;'><%=rb.getString("Kai")%></span>
								<span v-if="judgeActiveStatusFat(rowDataInfo.rf_status) == '2'" class='offOrOnStatusCls' style='margin-right: 5px;'><%=rb.getString("Kai")%></span>
								<span v-if="judgeActiveStatusFat(rowDataInfo.rf_status) == '3'" class='offStatusCls' style='margin-right: 5px;'><%=rb.getString("Guan")%></span>
								<el-popover title="<%= rb.getString("DuoXiaoQuZhuangTai")%>" popper-class="cellActivePopoverClass" trigger="click" width='300'>
									<span style="color:#4d84ff;" slot="reference">
										[ {{judgeActiveNumOrAllNumFat(rowDataInfo.rf_status,'active')}}/{{judgeActiveNumOrAllNumFat(rowDataInfo.rf_status,'all')}} ]
									</span>
									<div style="padding:10px;border-top:1px solid #E9E9E9;min-height:100px;display:flex;flex-wrap:wrap;">
										<div v-for="(item,index) in parseCellAndRfStatus(rowDataInfo.rf_status)" style="height:40px;display:flex;align-items: center;margin-left:10px;">
											Cell {{index+1}}:
											<span v-if=" item == 'on'" class="onStatusBoxCls" style='margin-left: 5px;'><%=rb.getString("Kai")%></span>
											<span v-if=" item == 'off'" class="offStatusBoxCls" style='margin-left: 5px;'><%=rb.getString("Guan")%></span>
										</div>
									</div>
								</el-popover>
							</div>
							<div v-else>
								<div v-if="rowDataInfo.rf_status == 'on'" class='iconFlexCls'>
									<span style='margin-right: 5px;'><%=rb.getString("Kai")%></span>
								</div>
								<div v-if="rowDataInfo.rf_status == 'off'" class='iconFlexCls'>
									<span class='offStatusCls' style='margin-right: 5px;'><%=rb.getString("Guan")%></span>
								</div>
							</div>
						</div>
                        <div class='commonItemBoxs'>
                            <span class='commonTemplateText12'>AMF Status</span>
                            <div v-if="rowDataInfo.product == 'BaiBNQ'">
                                <div v-if="['','NULL','null',null,undefined].includes(rowDataInfo.amf_status)">--</div>
                                <div v-if="!['','NULL','null',null,undefined].includes(rowDataInfo.amf_status)" style="display: flex;align-items: center;">
                                    <span v-html="amfStatusFmt(rowDataInfo.amf_status)" ></span>
                                    <el-popover title="All AMF Status" popper-class="moreStatusPopoverClass">
                                        <span style="color:#4d84ff;cursor:pointer;" slot="reference">
                                            [ {{parseParamsOnNum(rowDataInfo.amf_status)}}/{{parseParams(rowDataInfo.amf_status).length}} ]
                                        </span>
                                        <div class="moreStatusBoxCls">
                                            <i class="el-icon el-icon-close" style="position: absolute; top: -30px; right: 0px;" onclick="document.body.click()"></i>
                                            <div class="moreStatusItemCls" v-for="item in parseParams(rowDataInfo.amf_status)">
                                                <span v-if="item.Status == '1'" class="el-icon el-icon-status-MME greenIcon"></span>
                                                <span v-if="item.Status == '0'" class="el-icon el-icon-status-MME redIcon"></span> 
                                                <div class="moreStatusItemInfoCls">
                                                    <span>AMF IP : {{item.AmfIP1}}</span>
                                                    <span v-if="item.Status==='0'">AMF Status : <%= rb.getString("MMEWeiLianJie")%></span>
                                                    <span v-if="item.Status=='1'">AMF Status : <%= rb.getString("MMEYiLianJie")%></span>
                                                    <span v-if="item.Status=='2'">AMF Status : --</span>
                                                    <span v-if="item.Status==='' || item.Status===null">AMF Status : </span>
                                                    <span><%=rb.getString("PLMN")%> : {{item.PLMNID}}</span>
                                                </div>
                                            </div>
                                        </div>                                  
                                    </el-popover>
                                </div>                                   
                            </div>
                            <div v-else>--</div>
                        </div>
                        <div class='commonItemBoxs'>
                            <span class='commonTemplateText12'> </span>
                            <span class='commonSize12Texts'> </span>
                        </div>
					</div>
				</div>
			</div>
			<div style='width: 40%;'>
				<div class='commonContentBox' style="margin-bottom: 8px;">
					<div class="commonText14"><%=rb.getString("WangLuoSheZhi")%></div>
					<div class='commonFlexWarp' style='padding: 16px 0 0px;'>
						<div class='commonItemBoxs'>
							<span class='commonTemplateText12'><%=rb.getString("IPDiZhi")%></span>
							<span class='commonSize12Texts'>{{rowDataInfo.cell_ip}}</span>
						</div>
					</div>
				</div>
                <div class='commonContentBox'>
					<div class="commonText14"><%=rb.getString("WeiZhi")%></div>
					<div class='commonFlexWarp' style='padding: 16px 0 0px;'>
						<div class='commonItemBoxs'>
							<span class='commonTemplateText12'><%=rb.getString("JingDu")%></span>
							<span class='commonSize12Texts' v-if="rowDataInfo.gps_modify_flag != '1'">{{rowDataInfo.gps_longitude}}</span>
							<span class='commonSize12Texts' v-else>
								{{rowDataInfo.modify_longitude == undefined ? rowDataInfo.gps_longitude : rowDataInfo.modify_longitude}}
							</span>
						</div>
                        <div class='commonItemBoxs'>
							<span class='commonTemplateText12'><%=rb.getString("WeiDu")%></span>
							<span class='commonSize12Texts' v-if="rowDataInfo.gps_modify_flag != '1'">{{rowDataInfo.gps_latitude}}</span>
							<span class='commonSize12Texts' v-else>
								{{rowDataInfo.modify_latitude == undefined ? rowDataInfo.gps_latitude : rowDataInfo.modify_latitude}}
							</span>
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
</div>

<script>
var gnbOverviewVue = new Vue({
	el: '#gnbOverviewPage', 
	data() {
		return {
			rowDataInfo: [],
            currentRemarkLabel: '',
		};
	},
	computed: {},
	methods: {
		init(row){
			var vm = this;
			vm.rowDataInfo = row;
            vm.getCustomLabelData();
		},
        // 获取自定义label信息
        getCustomLabelData() {
            var vm = this;

            axios.post('${ctx}/cell/columnAlias/queryColumnAliasConfigs.action').then(function(response){
                var data = response.data || [];

                data.forEach(function(item){
                    if(item.columnName == 'remark'){
                        vm.currentRemarkLabel = item.columnAlias || 'Remark';
                    }
                });
            }).catch(function(error){});
        },
		ueCountsFormatter(rowData, value, rowIndex) {
			if(value === -1 || value === null || value === '' || value === undefined){
				return "<a style='color:#000000;text-decoration:none;cursor:default;' href='#'>--</a>"; 
			}else{
				return value
			}
		},
		
		// 判断小区状态 RF状态 显示  1 全部在线 2 部分在线 3 全部不在线
		judgeActiveStatusFat(value){
			var status = '1',
				states = (value+'').split(',');
			if((states.indexOf('0') >= 0  && states.indexOf('1') >= 0) || (states.indexOf('off') >= 0  && states.indexOf('on') >= 0)){
				status = '2'
			}else if((states.indexOf('0') >= 0  && states.indexOf('1') < 0) || (states.indexOf('off') >= 0  && states.indexOf('on') < 0)){
				status = '3'
			}else if((states.indexOf('0') < 0  && states.indexOf('1') >= 0) || (states.indexOf('off') < 0  && states.indexOf('on') >= 0)){
				status = '1'
			}
			return status
		},
		// 判断小区装填 RF状态  在线数 与 总数
		judgeActiveNumOrAllNumFat(value,type){
			var activeNum = [],
				allNum = [],
				states = (value+'').split(','),
				val = 0;
			states.map((item,index)=>{
				if(item == '1' || item == 'on'){
					activeNum.push(item)
				}
				allNum.push(item)
			})

			if(type == 'active'){
				val = activeNum.length
			}else{
				val = allNum.length
			}
			return val
		},
		// 生成 小区状态 RF状态 集合
		parseCellAndRfStatus(value){
			var states = (value+'').split(',');
			return states
		},
		cellStateFormatter(value, rowData, rowIndex){
			if (value == null) {
				return null;
			}
			else if (value == "1") {
				val = '<%= rb.getString("JiHuo")%>';
				value = "<div class=''>"+(val)+"</div>"
			}
			else if (value == "0") {
				//状态不一样展示的文字也不一样
				val = '<%= rb.getString("QuJiHuo")%>';
				value = "<div class='inactiveStatusItem'>"+(val)+"</div>"
			}
			return value;
		},
        parseParamsOnNum(str){
            var onNum = [],
                list = [];
            if(str) {
                list = eval('('+str+')');
            }else {
                list = [];
            }
            list.map((item,index)=>{
                if(item.Status == '1'){
                    onNum.push(item)
                }
            })
            return onNum.length
        },
        parseParams(str) {
            if(str) {
                return eval('('+str+')');
            }else {
                return [];
            }
        },
        amfStatusFmt(str){
            var amfList = eval('('+str+')');
            var amfStatus,textVal,value,hasDisconn=false,hasConn=false;
            amfList.map(function(item){
                if (item.Status == '1'){
                    hasConn = true;
                }else if(item.Status == '0'){
                    hasDisconn = true
                }
            })
            
            if (hasDisconn == true){
                if (hasConn == false) {
                    amfStatus ='offStatusCls';
                    textVal = '<%= rb.getString("MMEWeiLianJie")%>'
                }else{
                    amfStatus ='offOrOnStatusCls';
                    textVal = '<%= rb.getString("MMEYiLianJie")%>'
                }
            }else {
                amfStatus ='';
                textVal = '<%= rb.getString("MMEYiLianJie")%>'
            }
            
            value = "<span class='"+ amfStatus +"'>" + textVal +"</span>"
            return value;
        },
	},
	mounted() {
		eventBus.$off("gnb-data").$on("gnb-data",this.init)
	}
});

</script>
