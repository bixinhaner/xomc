<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<span class="el-icon el-icon-operation-settings" style="position:absolute;left:5px;z-index:99;box-shadow:none;bottom:-27px;" @click="columnStatus"></span>
<!-- 显示隐藏列 -->
<div class="showHideItem" id="showHideItemCpe">
    
    <div class="flex-ctn" id="sortAndShowColumnBoxCls" style="padding-left: 10px;flex-direction: row;border-bottom: 1px solid #e9e9e9;">
		<div style="padding: 0px 0 0 10px;height:100%;flex:3;overflow:auto;">
			<div class="select-all-cls">
				<span style="margin: 0;font-weight: bold;"><%=rb.getString("XuanZeLie")%></span>
			</div>
			<div class="select-all-cls" style="padding-left: 10px;">
				<el-checkbox :indeterminate="!colAll" v-model="colAll" @change="colAllChange"></el-checkbox> 
				<span><%=rb.getString("QuanXuan")%></span>
			</div>
			<div>
				<div class="select-all-cls">
					<i :class="{'el-icon':true,'el-icon-close1':expanded.device,'el-icon-open':!expanded.device}" @click="expanded.device = !expanded.device"></i>
					<el-checkbox :indeterminate="form.device.length<deviceCol.length" v-model="deviceAll" @change="deviceAllChange"></el-checkbox> 
					<span><%=rb.getString("SheBeiXinXi")%></span>
				</div>
				<el-checkbox-group v-show="expanded.device" class="col-group" v-model="form.device">
					<el-checkbox v-for="item in deviceCol" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}" :disabled="item.disabled">{{item.label}}</el-checkbox>
				</el-checkbox-group>
			</div>
			<div>
				<div class="select-all-cls">
					<i :class="{'el-icon':true,'el-icon-close1':expanded.lte,'el-icon-open':!expanded.lte}" @click="expanded.lte = !expanded.lte"></i>
					<el-checkbox :indeterminate="form.lte.length<lteCol.length" v-model="lteAll" @change="lteAllChange"></el-checkbox> 
					<span><%=rb.getString("LTEZhuangTai")%></span>
				</div>
				<el-checkbox-group v-show="expanded.lte" class="col-group" v-model="form.lte">
					<el-checkbox v-for="item in lteCol" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}" :disabled="item.disabled">{{item.label}}</el-checkbox>
				</el-checkbox-group>
			</div>
			<div>
				<div class="select-all-cls">
					<i :class="{'el-icon':true,'el-icon-close1':expanded.nr,'el-icon-open':!expanded.nr}" @click="expanded.nr = !expanded.nr"></i>
					<el-checkbox :indeterminate="form.nr.length<nrCol.length && form.nr.length>0" v-model="nrAll" @change="nrAllChange"></el-checkbox> 
					<span>NR Status</span>
				</div>
				<el-checkbox-group v-show="expanded.nr" class="col-group" v-model="form.nr">
					<el-checkbox v-for="item in nrCol" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}" :disabled="item.disabled">{{item.label}}</el-checkbox>
				</el-checkbox-group>
			</div>
			<div>
				<div class="select-all-cls">
					<i :class="{'el-icon':true,'el-icon-close1':expanded.lan,'el-icon-open':!expanded.lan}" @click="expanded.lan = !expanded.lan"></i>
					<el-checkbox :indeterminate="form.lan.length<lanCol.length" v-model="lanAll" @change="lanAllChange"></el-checkbox> 
					<span><%=rb.getString("LANZhuangTai")%></span>
				</div>
				<el-checkbox-group v-show="expanded.lan" class="col-group" v-model="form.lan">
					<el-checkbox v-for="item in lanCol" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}" :disabled="item.disabled">{{item.label}}</el-checkbox>
				</el-checkbox-group>
			</div>
			<div>
				<div class="select-all-cls">
					<i :class="{'el-icon':true,'el-icon-close1':expanded.location,'el-icon-open':!expanded.location}" @click="expanded.location = !expanded.location"></i>
					<el-checkbox :indeterminate="form.location.length<locationCol.length && form.location.length>0" v-model="locationAll" @change="locationAllChange"></el-checkbox> 
					<span><%=rb.getString("WeiZhi")%></span>
				</div>
				<el-checkbox-group v-show="expanded.location" class="col-group" v-model="form.location">
					<el-checkbox v-for="item in locationCol" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}">{{item.label}}</el-checkbox>
				</el-checkbox-group>
			</div>
		</div>

		<div style="border-left: 1px solid #e9e9e9;height:100%;flex:1;">
			<div class="select-all-cls" style="margin: 10px; padding: 0px; border-bottom: 1px solid #e9e9e9;">
				<el-checkbox v-if="false" :indeterminate="!dragAll" v-model="dragAll" @change="dragAllChange"></el-checkbox> 
				<span v-if="false"><%=rb.getString("QuanXuan")%></span>
				<span style="padding-bottom: 10px;margin-left: 0px;font-weight: bold;"><%=rb.getString("LiePaiXu")%></span>
			</div>
			<el-checkbox-group v-model="dragCol">
				<draggable
					class="list-group"
					v-model="columns"
					v-bind="dragOptions">
					<transition-group type="transition" :name="!drag? 'flip-list':null">
						<div v-for="(col,idx) in columns" :key="col.field" class="list-group-item" v-if="showCols.includes(col.field)">
							<span class="el-checkbox__label" style="border-bottom: 1px dashed #e9e9e9;min-width: 220px;color: #606266;">
								{{col.label}}
								
								<i v-if="!col.disabled" class="el-icon el-icon-close" style="zoom: 0.6;float: right; margin-top: 6px;" @click="clickColumnLabel(col.label)"></i>
							</span>
							<el-checkbox v-if="false" :label="col.field" :key="col.field" :disabled="col.disabled">{{col.label}} </el-checkbox>
						</div>
					</transition-group>
				</draggable>
			</el-checkbox-group>
		</div>
    </div>
    <div class="windowButtonGroup" style="float:none !important;padding:20px 0 20px 30px;position:relative;z-index:321;">
        <a class="linkbutton linkbutton_trend" @click="columnConfig"><span><%=rb.getString("QueDing")%></span></a>
        <a class="linkbutton linkbutton_nowanna" @click="closeConfig"><span><%=rb.getString("QuXiao")%></span></a>
    </div>
</div>
<div v-if="isExisted && isAdmin" class="fixed-right-msg">
	<div v-if="msgExtend" style="margin-right: 20px;line-height:26px;">
		<span style="padding: 0px 10px;">{{collectSn}}</span>
		<span style="padding: 0px 5px;" v-if="taskTime==''">
			<i class="el-icon el-icon-status-yes" style="font-size: 12px;"></i>
			<%=rb.getString("ChengGong")%>
		</span>
		<span v-if="taskTime!=''" style="display: inline-block;padding: 2px 30px;background: #4d84ff;border-radius: 2px;margin: 0px 5px 2px 5px;"></span>
		<span v-if="taskTime!=''" style="border: 1px solid #e3e3e3;border-radius: 3px;padding: 2px 4px;">
			<span style="cursor: pointer;" @click="stopCollect">
				<i style="padding: 4px;background: red;height: 0px;display: inline-block;border-radius: 3px;"></i>
				<%=rb.getString("TingZhi")%>
			</span>
			<span style="margin-left: 5px;">
				{{taskTime}}
			</span>
		</span>
		<span style="margin-left: 20px;">
			<a class="collect-bt" @click="viewMsg"><%=rb.getString("ChaKan")%></a>
			<a class="collect-bt" @click="downloadMsg"><%=rb.getString("XiaZai")%></a>
			<a class="collect-bt" @click="clearMsg"><%=rb.getString("QingChu")%></a>
		</span>
	</div>
	<div @click="msgExtend = !msgExtend" class="foldBtnCls">
		<span v-if="msgExtend" class="el-icon el-icon-common-query-up"></span>
		<span v-if="!msgExtend" class="el-icon el-icon-common-query-down"></span>
	</div>
</div>
<div id="tableHeadQuery" class="tableHeadQueryBoxCls">
	<div class="headQueryBox">
		<div class="queryGroup">
			<el-input v-model="search_text" @keyup.enter.native="query" @focus="queryInputFocus" @blur="queryInputBlur" :placeholder='placeholderText' style="width:260px;"></el-input>
			<i @click='query' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
		</div>
	</div>
	<div v-for="(item,index) in advancedQueryItemList">
		<div v-if="item.type == 'checkbox' && item.isShow" style="margin-right:10px;">
			<el-popfilter
				:label='item.label'
				v-model="item.checkedItemList"
				:list="item.options"
				:visible.sync="item.isShow"
				:closable="true"
				@check-change="advanceQuery(item.type,item.value,item.checkedItemList)"
				@close="checkItemDel(item)">
			</el-popfilter>
		</div>
		<div v-if="item.type == 'select' && item.isShow" style="margin-right:10px;">
			<el-popfilter
				type="single"
				:label='item.label'
				v-model="item.selectVal"
				:list="item.options"
				:visible.sync="item.isShow"
				:closable="true"
				@check-change="advanceQuery(item.type,item.value,item.selectVal)"
				@close="checkItemDel(item)">
			</el-popfilter>
		</div>
		<div v-if="item.type == 'filter'" style="margin-right:10px;">
			<div class="advancedQueryItemBox" style="background: #FFF;">
				<el-popover :ref="'popover-'+item.value" trigger="click" placement="bottom-start"  @show="checkPopoverShow(item)" @hide="checkPopoverHide(item,'')">
					<div class="checkPopoverBoxCls">
						<el-checkbox-group v-model="item.checkedItemList" @change="handleCheckedChange(item)">
							<el-checkbox v-for=" items in item.options" :key="items.value" :label="items.value">{{items.label}}</el-checkbox>
						</el-checkbox-group>
						<div class="buttonGroup">
							<el-button size="mini" type="primary" @click="checkPopoverSubmit(item)"><%=rb.getString("QueDing")%></el-button>
							<el-button size="mini" @click="checkPopoverHide(item,'del')"><%=rb.getString("QuXiao")%></el-button>
						</div>	
					</div>
					<div slot="reference" class="ItemAndIconBoxCls">
						<i class="el-icon el-icon-filterAdd"></i>
						{{item.label}}
					</div>
				</el-popover>
			</div>
		</div>
	</div>
	<div class="advancedQueryItemBox"  style="background: #FFF;" @click="clearFilterClick">
		<%=rb.getString("QingKongShaiXuan")%>
	</div>
	<div class="advancedQueryItemBox" style="background: #FFF;position:absolute;right:10px;top:7px;" @click="timeLockChange">
		<i :class="{'el-icon':true, 'placeholder-bt':true, 'placeholder-bt':true, 'last-bt':true, 'el-icon-monitor-lock':timeLocked, 'el-icon-monitor-refresh':!timeLocked}" 
			style="cursor:pointer;position:relative;margin-right:5px;"></i>
		<span v-show="timeLocked"><%=rb.getString("SuoDingShuaXin")%></span>
		<span v-show="!timeLocked"><%=rb.getString("DingShiShuaXin")%></span>
	</div>
</div>
<el-dialog :visible.sync="collectInfoShow" top="10vh">
	<div slot="title">
		<span class="el-dialog__title"><%=rb.getString("XinXi")%></span>
		<i class="el-icon el-icon-operation-export" style="position: absolute;right: 45px;top: 9px;" @click="downloadMsg"></i>
	</div>
	<el-input v-model="collectContent" type="textarea" rows="25" readonly class="none-border"></el-input>
</el-dialog>

<script>
    var queryCpeVue = new Vue({
		el: '#toolbar_tableHomeCpeList',
		data() {

			return {
				placeholderText:'<%=rb.getString("QingShuRu")%>',
				advancedQueryItemList:[
					{
						type:'checkbox',
						isShow:true,
						isIndeterminate:false,
						checkAll:false,
						popoverShow:false,
						checkedItemList:[],
						oldCheckedItemList:[],
						label:'<%=rb.getString("ZaiXianZhuangTai") %>',
						options:[
							{label:'<%=rb.getString("LianJieZhengChang")%>',value:"1"},
							{label:'<%=rb.getString("LianJieDuanKai")%>',value:"0"},
							{label:'<%=rb.getString("TongBuZhong")%>',value:"3"},
							{label:'<%=rb.getString("TongBuShiBai")%>',value:"2"}
						],
						value:'connection_status',
					},
					{
						type:'checkbox',
						isShow:true,
						isIndeterminate:false,
						checkAll:false,
						popoverShow:false,
						checkedItemList:[],
						oldCheckedItemList:[],
						label:'<%=rb.getString("ChanPinXingHao") %>',
						options:[],
						value:'product_model',
					},
					{
						type:'select',
						isShow:false,
						popoverShow:false,
						selectVal:'',
						label:'<%=rb.getString("ChanPinLeiXing") %>',
						options:[],
						value:'cpeMonitorModule',
					},
					{
						type:'checkbox',
						isShow:false,
						isIndeterminate:false,
						checkAll:false,
						popoverShow:false,
						checkedItemList:[],
						oldCheckedItemList:[],
						label:'<%=rb.getString("SoftwareVersion") %>',
						options:[],
						value:'software_version',
					},
					{
						type:'checkbox',
						isShow:false,
						isIndeterminate:false,
						checkAll:false,
						popoverShow:false,
						checkedItemList:[],
						oldCheckedItemList:[],
						label:'<%=rb.getString("SheBeiZu") %>',
						options:[],
						value:'group_id',
					},
					{
						type:'filter',
						popoverShow:false,
						checkedItemList:['connection_status','product_model'],
						oldCheckedItemList:['connection_status','product_model'],
						label:'<%=rb.getString("TianJiaShuaiXuan") %>',
						options:[
							{label:'<%=rb.getString("ZaiXianZhuangTai") %>',value:"connection_status"},
							{label:'<%=rb.getString("ChanPinXingHao")%>',value:"product_model"},
							{label:'<%=rb.getString("ChanPinLeiXing")%>',value:"cpeMonitorModule"},
							{label:'<%=rb.getString("SoftwareVersion")%>',value:"software_version"},
							{label:'<%=rb.getString("SheBeiZu")%>',value:"group_id"},
						],
						value:'add_filter',
					}

				],

				isExisted: false,
				collectDeviceCode: '',
				collectSn: '',
				taskTime: '',
				msgExtend: false,
				collectInfoShow: false,
				collectContent: '',
				
				dragCol: [
					// 'SERIAL_NUMBER',
					// 'CPE_NAME',
					'IMSI',
					'MACADDRESS',
					'IPADDRESS',
					'cpe_model',
					'MODEL_NAME',
					'SOFTWARE_VERSION',
					'group_name',
					'HOST_NAME',
					'CELL_IDENTITY',
					'PCI'
				],
				drag: false,
				columns: [
					// {field: 'SERIAL_NUMBER', label: '<%=rb.getString("CPEXuLieHao")%>',sortable: true ,disabled: true, width: 180},
					// {field: 'CPE_NAME', label: '<%=rb.getString("CPEName")%>',sortable: true ,disabled: true, width: 120},
					{field: 'IMSI', label: 'IMSI',sortable: true ,disabled: true, width: 150},
					{field: 'MACADDRESS', label: 'MAC',sortable: true ,disabled: true, width: 150},
					{field: 'IPADDRESS', label: 'IP',sortable: true ,disabled: true, width: 120},
					{field: 'cpe_model', label: '<%=rb.getString("SheBeiXingHao")%>',disabled: true, width: 120},
					{field: 'MODEL_NAME', label: '<%=rb.getString("ChanPinXingHao")%>',sortable: true ,disabled: true, width: 160},
					{field: 'SOFTWARE_VERSION', label: '<%=rb.getString("RuanJianBanBen")%>',sortable: true ,disabled: true, width: 160},
					{field: 'group_name', label: '<%=rb.getString("SheBeiZu")%>',sortable: true ,disabled: true, width: 150},
					{field: 'HOST_NAME', label: '<%=rb.getString("HostName")%>',sortable: true ,disabled: true, width: 120},
					{field: 'CELL_IDENTITY', label: 'ECI',sortable: true ,disabled: true, width: 120},
					{field: 'SCANMODE', label: '<%=rb.getString("SaoMiaoFangShi")%>', width: 150},
					{field: 'PCI', label: 'PCI',sortable: true ,disabled: true, width: 80},
					{field: 'LGW_IP', label: '<%=rb.getString("LgwIPAddress")%>', width: 130},
					{field: 'LGW_MAC', label: '<%=rb.getString("LgwMacAddress")%>',sortable: true, width: 170},
					{field: 'UL_MCS', label: 'UL_MCS',sortable: true, width: 75},
					{field: 'DL_MCS', label: 'DL_MCS',sortable: true, width: 75},
					{field: 'lte_connection_time', label: '<%=rb.getString("LTEGengXinShiJian")%>',sortable: true, width: 100},
					{field: 'DL_BLER', label: 'DL BLER',sortable: true, width: 75},
					{field: 'RSRP0', label: 'RSRP1',sortable: true, width: 80},
					{field: 'RSRP1', label: 'RSRP2',sortable: true, width: 80},
					{field: 'CINR0', label: 'CINR1',sortable: true, width: 80},
					{field: 'CINR1', label: 'CINR2',sortable: true, width: 80},
					{field: 'CPE_SINR', label: 'SINR',sortable: true, width: 80},
					{field: 'DL_CURRENT_DATARATE', label: '<%=rb.getString("CPEXiaXingTunTuLiang")%> (Mbps)',sortable: true, width: 180},
					{field: 'UL_CURRENT_DATARATE', label: '<%=rb.getString("CPEShangXingTunTuLiang")%> (Mbps)',sortable: true, width: 180},
					{field: 'UPTIME', label: '<%=rb.getString("YunXingShiJian")%>',sortable: true, width: 140},
					{field: 'first_online_time', label: '<%=rb.getString("DiYiCiLianJieShiJian")%>',sortable: true, width: 140},
					{field: 'LASTINFORMTIME', label: '<%=rb.getString("ShangCiLianJieShiJian")%>',sortable: true, width: 140},
					{field: 'PRODUCT', label: '<%=rb.getString("ChanPinLeiXing")%>',sortable: true, width: 80},
					{field: 'TX_POWER', label: '<%=rb.getString("CPETxPower")%>',sortable: true, width: 90},
					{field: 'DL_EARFCN', label: '<%=rb.getString("PinDian")%>', width: 60},
					{field: 'BANDWIDTH', label: '<%=rb.getString("DaiKuan")%>(MHz)', width: 130},
					{field: 'MCC', label: 'MCC',sortable: true, width: 80},
					{field: 'MNC', label: 'MNC',sortable: true, width: 80},
					{field: 'longitude', label: '<%=rb.getString("JingDu")%>', width: 90},
					{field: 'latitude', label: '<%=rb.getString("WeiDu")%>', width: 90},
					{field: 'height', label: '<%=rb.getString("GaoDu")%>', width: 90},
					{field: 'distance', label: '<%=rb.getString("JuLi")%>', width: 90},
					{field: 'module_name', label: '<%=rb.getString("MoKuaiMingCheng")%>', width: 150},
					{field: 'module_version', label: '<%=rb.getString("MoKuaiBanNen")%>', width: 150},
                    {field: 'MARKET_NAME', label: '<%=rb.getString("ChanPinMingCheng")%>', width: 150},
					{field: 'IMEI', label: 'IMEI', width: 150},

					{field: 'NR_BAND', label: 'NR-Band', width: 80},
					{field: 'NR_BANDWIDTH', label: 'NR-<%=rb.getString("DaiKuan")%>(MHz)', width: 160},
					{field: 'NR_PCI', label: 'NR-PCI', width: 80},
					{field: 'NR_EARFCN', label: 'NR-<%=rb.getString("PinDian")%>', width: 90},
					{field: 'NR_PLMN', label: 'NR-PLMN', width: 90},
					{field: 'NR_CELL_ID', label: 'NR-Cell ID', width: 120},
					{field: 'NR_DL_FREQUENCY', label: 'NR-DL_Frequency', width: 150},
					{field: 'NR_UL_FREQUENCY', label: 'NR-UL_Frequency', width: 150},
					{field: 'NR_CINR', label: 'NR-CINR', width: 90},
					{field: 'NR_SINR', label: 'NR-SINR', width: 90},
					{field: 'NR_RSRQ', label: 'NR-RSRQ', width: 90},
					{field: 'NR_RSRP', label: 'NR-RSRP', width: 90},
				],

				deviceCol: [
					{code: 'SERIAL_NUMBER', label: '<%=rb.getString("CPEBianMa")%>',disabled: true},
					{code: 'CPE_NAME', label: '<%=rb.getString("CPEName")%>',disabled: true},
					{code: 'MODEL_NAME', label: '<%=rb.getString("ChanPinXingHao")%>',disabled: true},
					{code: 'PRODUCT', label: '<%=rb.getString("ChanPinLeiXing")%>'},
					{code: 'SOFTWARE_VERSION', label: '<%=rb.getString("CPEVersion")%>',disabled: true},
					{code: 'UPTIME', label: '<%=rb.getString("YunXingShiJian")%>'},
					{code: 'first_online_time', label: '<%=rb.getString("DiYiCiLianJieShiJian")%>'},
					{code: 'LASTINFORMTIME', label: '<%=rb.getString("ShangCiLianJieShiJian")%>'},
					{code: 'MCC', label: 'MCC'},
					{code: 'MNC', label: 'MNC'},
					{code: 'group_name', label: '<%=rb.getString("SheBeiZu")%>',disabled: true},
					{code: 'module_name', label: '<%=rb.getString("MoKuaiMingCheng")%>'},
					{code: 'module_version', label: '<%=rb.getString("MoKuaiBanNen")%>'},
					{code: 'LGW_IP', label: '<%=rb.getString("LgwIPAddress")%>'},
                    {code: 'MARKET_NAME', label: '<%=rb.getString("ChanPinMingCheng")%>', width: 150},
					{code: 'IMEI', label: 'IMEI', width: 100},
					{code: 'cpe_model', label: '<%=rb.getString("SheBeiXingHao")%>', width: 120,disabled: true}
				],
				lteCol: [
					{code: 'IMSI', label: 'IMSI',disabled: true},
					{code: 'SCANMODE',label:'<%=rb.getString("SaoMiaoFangShi")%>'},
					{code: 'PCI', label: 'PCI',disabled: true},
					{code: 'HOST_NAME', label: '<%=rb.getString("HostName")%>',disabled: true},
					{code: 'CELL_IDENTITY', label: 'ECI',disabled: true},
					{code: 'DL_EARFCN', label: '<%=rb.getString("PinDian")%>'},
					{code: 'BANDWIDTH', label: '<%=rb.getString("DaiKuan")%>(MHz)'},
					{code: 'CINR0', label: 'CINR1'},
					{code: 'CINR1', label: 'CINR2'},
					{code: 'CPE_SINR', label: 'SINR'},
					{code: 'DL_CURRENT_DATARATE', label: '<%=rb.getString("CPEXiaXingTunTuLiang")%>'},
					{code: 'UL_CURRENT_DATARATE', label: '<%=rb.getString("CPEShangXingTunTuLiang")%>'},
					{code: 'TX_POWER', label: '<%=rb.getString("CPETxPower")%>'},
					{code: 'RSRP0', label: 'RSRP1'},
					{code: 'RSRP1', label: 'RSRP2'},
					{code: 'UL_MCS', label: 'UL_MCS'},
					{code: 'DL_MCS', label: 'DL_MCS'},
					{code:'lte_connection_time',label:'<%=rb.getString("LTEGengXinShiJian")%>'},
					{code: 'DL_BLER', label: 'DL BLER'},
				],
				nrCol: [
					{code: 'NR_BAND', label: 'NR-Band'},
					{code: 'NR_BANDWIDTH', label: 'NR-<%=rb.getString("DaiKuan")%>(MHz)'},
					{code: 'NR_PCI', label: 'NR-PCI'},
					{code: 'NR_EARFCN', label: 'NR-<%=rb.getString("PinDian")%>'},
					{code: 'NR_PLMN', label: 'NR-PLMN'},
					{code: 'NR_CELL_ID', label: 'NR-Cell ID'},
					{code: 'NR_DL_FREQUENCY', label: 'NR-DL_Frequency'},
					{code: 'NR_UL_FREQUENCY', label: 'NR-UL_Frequency'},
					{code: 'NR_CINR', label: 'NR-CINR'},
					{code: 'NR_SINR', label: 'NR-SINR'},
					{code: 'NR_RSRQ', label: 'NR-RSRQ'},
					{code: 'NR_RSRP', label: 'NR-RSRP'},
				],
				lanCol: [
					{code: 'MACADDRESS', label: '<%=rb.getString("CPEMacAddress")%>',disabled: true},
					{code: 'IPADDRESS', label: '<%=rb.getString("CPEIPAddress")%>',disabled: true},
					{code: 'LGW_MAC', label: '<%=rb.getString("LgwMacAddress")%>'}
				],
				locationCol: [
					{code: 'longitude', label: '<%=rb.getString("JingDu")%>'},
					{code: 'latitude', label: '<%=rb.getString("WeiDu")%>'},
					{code: 'height', label: '<%=rb.getString("GaoDu")%>'},
					{code: 'distance', label: '<%=rb.getString("JuLi")%>'},
				],
				form: {
					device: ['SERIAL_NUMBER','CPE_NAME','MODEL_NAME','SOFTWARE_VERSION','group_name','LGW_IP','cpe_model'],
					lte: ['IMSI','PCI','HOST_NAME','CELL_IDENTITY'],
					nr: [],
					lan: ['MACADDRESS','IPADDRESS'],
					location: []
				},
				expanded: {
					device: true,
					lte: true,
					lan: true,
					nr: true,
					location: true
				},
				queryOptions:[
					{label:'<%=rb.getString("QuanBu")%>',value:""},
					{label:'<%=rb.getString("CPEBianMa")%>',value:"serial_number"},
					{label:'<%=rb.getString("CPEName")%>',value:"CPE_NAME"},
					{label:'<%=rb.getString("HostName")%>',value:"HOST_NAME"},
					{label:'IMSI',value:"IMSI"},
					{label:'<%=rb.getString("IPDiZhi")%>',value:"ipaddress"},
					{label:'<%=rb.getString("MACDiZhi")%>',value:"macaddress"},
					{label:'ECI',value:"CELL_IDENTITY"},
					{label:'PCI',value:"pci"},
					{label:'LGW IP',value:"lgw_ip"},
					{label:'LGW MAC',value:"lgw_mac"}
					],
				
				search_text: '',
				showSelectButton:false,
				operator_code:operator_code,
				queryParams:{
					TimeZone : timeZone
				},
				timeLocked:true
			};
		},
		computed: {
			isAdmin() {
				return is_super_user == 'true'
			},
			dragOptions() {

				return {
					animation: 200,
					group: 'description',
					disabled: false,
					ghostClass: 'ghost'
				};
			},
			dragAll() {
				var vm = this;
				return vm.dragCol.length == vm.columns.length;
			},
			colAll() {
				var vm = this;
				return vm.deviceAll && vm.lteAll && vm.nrAll && vm.lanAll && vm.locationAll;
			},
			deviceAll() {
				var vm = this;
				return vm.deviceCol.length == vm.form.device.length;
			},
			lteAll() {
				var vm = this;
				return vm.lteCol.length == vm.form.lte.length;
			},
			nrAll() {
				var vm = this;
				return vm.nrCol.length == vm.form.nr.length;
			},
			lanAll() {
				var vm = this;
				return vm.lanCol.length == vm.form.lan.length;
			},
			locationAll() {
				var vm = this;
				return vm.locationCol.length == vm.form.location.length;
			},
			showCols() {
				var vm = this;
				return vm.form.device.concat(vm.form.lte).concat(vm.form.nr).concat(vm.form.lan).concat(vm.form.location);
				//return vm.dragCol;
			},
		},
		methods: {
			timeLockChange() {
				var vm = this;
				
				vm.timeLocked = !vm.timeLocked;
			},
			// 模糊搜索
			query(){
				var vm = this;
				vm.queryParams.search_text = this.search_text;
				vm.queryParams['like_fields'] = 'HOST_NAME,CPE_NAME,macaddress,IMSI,serial_number,ipaddress,CELL_IDENTITY,pci,lgw_ip,lgw_mac';
				Object.assign(cpevm.queryParams, vm.queryParams);
			},
			// 搜索域聚焦事件
			queryInputFocus(){
				var vm = this;
				vm.placeholderText = '<%=rb.getString("CPEBianMa")%>/ <%=rb.getString("CPEName")%>/ <%=rb.getString("HostName")%>/ IMSI/ IP/ MAC/ ECI/ PCI/ LGW IP/ LGW MAC';
			},
			// 搜索域失焦事件
			queryInputBlur(){
				var vm = this;
				vm.placeholderText = '<%=rb.getString("QingShuRu")%>';
			},
			// 高级查询 确定事件
			advanceQuery(type,paramsItem,value){
				var vm = this,
					params ={};
				if(type == 'select'){
					params[paramsItem] = value;
				}else{
					params[paramsItem] = value.join(',');
				}
				Object.assign(cpevm.queryParams, params);
			},
			// 高级搜索下拉全选事件
			handleCheckedAllChange(item){
				var vm = this,
					allList = [];
				item.options.map((items)=>{
					if(items.value){
						allList.push(items.value)
					}
				})
				vm.advancedQueryItemList.map((items)=>{
					if(item.value == items.value){
							items.checkedItemList = items.checkAll? allList : [];
							items.isIndeterminate = false;
					}
				})
			},
			// 高级搜索下拉单选事件
			handleCheckedChange(item){
				var vm = this,
					checkCount = item.checkedItemList.length,
					allCount = item.options.length;
				
				vm.advancedQueryItemList.map((items)=>{
					if(item.value == items.value){
						if(items.type == 'checkbox'){
							items.checkAll = checkCount === allCount;
							items.isIndeterminate = checkCount > 0 && checkCount < allCount;
						}
					}
				})
			},
			// 高级搜索下拉弹窗 展开事件
			checkPopoverShow(item){
				var vm = this;
				vm.advancedQueryItemList.map((items)=>{
					if(item.value == items.value){
						items.popoverShow = true;
					}
				})
			},
			// 高级搜索下拉弹窗 收起事件
			checkPopoverHide(item,types){
				var vm = this,
					checkCount = 0,
					allCount = item.options.length;
				if(item.type != 'select'){
					checkCount = item.oldCheckedItemList.length
				}
				vm.advancedQueryItemList.map((items)=>{
					if(item.value == items.value){
						items.popoverShow = false;
						if(items.type != 'select'){
							items.checkedItemList = items.oldCheckedItemList;
						}
						if(items.type == 'checkbox'){
							items.checkAll = checkCount === allCount;
							items.isIndeterminate = checkCount > 0 && checkCount < allCount;
						}
					}
				})
				if(types == 'del'){
					document.body.click();
				}
			},
			// 高级搜索下拉弹窗 选择提交
			checkPopoverSubmit(item){
				var vm = this,
					params ={};
				if(item.type == 'checkbox'){
					vm.advancedQueryItemList.map((items)=>{
						if(item.value == items.value){
							items.oldCheckedItemList = items.checkedItemList;
							if(items.checkAll){
								params[items.value] = '';
							}else{
								params[items.value] = items.checkedItemList.join(',');
							}
						}
					})
				}else{
					vm.advancedQueryItemList.map((items)=>{
						if(item.value == items.value){
							items.oldCheckedItemList = items.checkedItemList;
						}else{
							var result = item.checkedItemList.includes(items.value);
							if(!result){
								items.isShow = false;
								items.checkedItemList = [];
								items.oldCheckedItemList = [];
								items.checkAll = false;
								items.isIndeterminate = false;
								params[items.value] = '';
							}else{
								items.isShow = true;
							}
						}
					})
				}
				
				Object.assign(cpevm.queryParams, params);
				document.body.click();
			},
			// 筛选项删除事件
			checkItemDel(item){
				var vm = this,
					params ={};
				vm.advancedQueryItemList.map((items)=>{
					if(item.value == items.value){
						items.isShow = false;
						if(items.type != 'select'){
							items.checkedItemList = [];
							items.oldCheckedItemList = [];
							items.checkAll = false;
							items.isIndeterminate = false;
						}else{
							items.selectVal = '';
						}
						params[items.value] = '';
					}
					if(items.value == 'add_filter'){
						var idx = items.checkedItemList.indexOf(item.value),
							oldIdx = items.oldCheckedItemList.indexOf(item.value);
						if(idx>-1){
							items.checkedItemList.splice(idx,1)
						}
						if(oldIdx>-1){
							items.oldCheckedItemList.splice(idx,1)
						}
					}

				})
				Object.assign(cpevm.queryParams, params);
			},
			//单选 选择事件
			selectChangeClick(value,item){
				var vm = this,
					params ={};
				
				vm.advancedQueryItemList.map((items)=>{
					if(item.value == items.value){
						items.selectVal = value;
						params[items.value] = value;
					}
				})
				Object.assign(cpevm.queryParams, params);
				document.body.click();
			},
			// 清除筛选
			clearFilterClick(){
				var vm = this,
					params = {};
				vm.advancedQueryItemList.map((items)=>{
					if(items.isShow && items.isShow== true ){
						if(items.type == 'checkbox'){
							items.checkedItemList = [];
							items.oldCheckedItemList = [];
							items.checkAll = false;
							items.isIndeterminate = false;
						}else if(items.type == 'select'){
							items.selectVal = '';
						}
						if(items.type == 'checkbox' || items.type == 'select'){
							params[items.value] = '';
						}
					}
				})
				Object.assign(cpevm.queryParams, params);
				document.body.click();
			},
			queryLatestInfo() {
				var vm = this,
					params = {
						type: 'cpe',
						operatorCode: operatorCodeGloab
					};

				axios.post('${ctx}/trace/queryLatestMessageTraceDeviceInfo.action', stringify(params)).then(function(res){
					var data = res.data;

					if(data) {
						vm.collectSn = data.serialNumber;
						vm.collectDeviceCode = data.deviceCode;

						if(data.status == '0' && data.remainTime) {
							vm.startInterval(data.remainTime);
						}else if(data.status == '1'){
							vm.taskTime = '';
							vm.startInterval('00:01');
						}

						if(['',undefined].includes(data.status) && ['',undefined].includes(data.serialNumber)) {
							vm.isExisted = false;
						}else {
							vm.isExisted = true;
						}
					}
				});
			},
			viewMsg() {
				var vm = this,
					params = {
						serialNumber: vm.collectSn,
						type: 'cpe'
					};
						
				vm.collectInfoShow = true;

				axios.post('${ctx}/trace/queryMessageTraceInfo.action', stringify(params)).then(function(res){
					var data = res.data;

					if(data) {
						vm.collectContent = data;
						vm.collectInfoShow = true;
					}
				});
			},
			downloadMsg() {
				var vm = this,
					params = {
						serialNumber: vm.collectSn,
						type: 'cpe',
						timeZone: timeZone
					};

				exportByForm('${ctx}/trace/downLoadMessageTraceInfo.action',params);
			},
			clearMsg() {
				var vm = this,
					params = {
						deviceCode: vm.collectDeviceCode,
						serialNumber: vm.collectSn,
						type: 'cpe',
						operatorCode: operatorCodeGloab
					};

				axios.post('${ctx}/trace/clear.action', stringify(params)).then(function(res){
					var data = res.data;

					if(data.success == true) {
						vm.$message.success('<%=rb.getString("ChengGong")%>');
						vm.queryLatestInfo();
					}else {
						vm.$message.error(data.message);
					}
				});
			},
			stopCollect() {
				var vm = this,
					row = cpevm.selectedRow || {},
					params = {
						deviceCode: row.CPE_CODE,
						serialNumber: row.SERIAL_NUMBER,
						type: 'cpe',
						operatorCode: operatorCodeGloab
					};

				axios.post('${ctx}/trace/stop.action', stringify(params)).then(function(res){
					var data = res.data;

					if(data.success == true) {
						vm.$message.success('<%=rb.getString("ChengGong")%>');
						vm.queryLatestInfo();
					}else {
						vm.$message.error(data.message);
					}
				});
			},
			startInterval(time) {
				var vm = this,
					list = time.split(':'),
					totalTime = list[0]*60 + list[1]*1;

				clearInterval(cpeCollectInterval);
				
				cpeCollectInterval = setInterval(function() {
					totalTime -= 1;
					vm.taskTime = vm.formaterTime(Math.floor(totalTime/60)) + ':' + vm.formaterTime(totalTime%60);

					if(totalTime<1) {
						clearInterval(cpeCollectInterval);
						vm.taskTime = '';
					}
				},1000);
			},
			formaterTime(num) {

				return num < 10? '0'+num : num;
			},
			
			clickColumnLabel(label) {
				$('.el-checkbox__label:contains('+label+')').click();
			},
			dragAllChange(val) {
				var vm = this,
					fields = vm.columns.map(function(item){
						return item.field;
					}),
					filters = vm.columns.filter(function(item){
						return item.disabled;
					}).map(function(item){
						return item.field;
					});
				
				vm.dragCol = val?fields:filters;
			},
			colAllChange(val) {
				var vm = this;

				vm.deviceAllChange(val);
				vm.lteAllChange(val);
				vm.nrAllChange(val);
				vm.lanAllChange(val);
				vm.locationAllChange(val);
			},
			deviceAllChange(val) {
				var vm = this,
					fields = vm.deviceCol.map(function(item){
						return item.code;
					}),
					filters = vm.deviceCol.filter(function(item){
						return item.disabled;
					}).map(function(item){
						return item.code;
					});
				
				vm.form.device = val?fields:filters;
			},
			lteAllChange(val) {
				var vm = this,
					fields = vm.lteCol.map(function(item){
						return item.code;
					}),
					filters = vm.lteCol.filter(function(item){
						return item.disabled;
					}).map(function(item){
						return item.code;
					});
				
				vm.form.lte = val?fields:filters;
			},
			nrAllChange(val) {
				var vm = this,
					fields = vm.nrCol.map(function(item){
						return item.code;
					}),
					filters = vm.nrCol.filter(function(item){
						return item.disabled;
					}).map(function(item){
						return item.code;
					});
				
				vm.form.nr = val?fields:filters;
			},
			lanAllChange(val) {
				var vm = this,
					fields = vm.lanCol.map(function(item){
						return item.code;
					}),
					filters = vm.lanCol.filter(function(item){
						return item.disabled;
					}).map(function(item){
						return item.code;
					});
				
				vm.form.lan = val?fields:filters;
			},
			locationAllChange(val) {
				var vm = this,
					fields = vm.locationCol.map(function(item){
						return item.code;
					}),
					filters = vm.locationCol.filter(function(item){
						return item.disabled;
					}).map(function(item){
						return item.code;
					});
				
				vm.form.location = val?fields:filters;
			},
			closeConfig() {
				$(".showHideItem").slideUp(500);
			},
			columnConfig() {
				var vm = this,
					url = '${ctx}/cell/CPE/cpeColumnConfig.action',
					sortCol = [];

				vm.columns.map(function(item){
					sortCol.push(item.field);
				});

				vm.configTable();

				url = '${ctx}/system/column/setting/insert.action';
				// 保存显示列
				var params = {
						pageName: '2',
						showColumn: vm.showCols.join(','),
						sortColumn: sortCol.join(',')
					};

				axios.post(url, params).then(function(res){
					cpevm.columns.splice(0, cpevm.columns.length);
					vm.columns.map(function(col){
						cpevm.columns.push(Object.assign({},col));
					});
					
					var tbStates = cpevm.$refs.list.$refs.ctableInner.store.states,
						sortCodes = sortCol,
						storeCols = tbStates.columns;
	
					tbStates._columns = storeCols.sort(function(n, m) {
						var idxn = sortCodes.indexOf(n.property)==-1?100:sortCodes.indexOf(n.property),
							idxm = sortCodes.indexOf(m.property)==-1?100:sortCodes.indexOf(m.property);
	
						if( [undefined,'cpe_operation','CONNECTION_STATUS',,'SERIAL_NUMBER','CPE_NAME'].includes(n.property) ) {
							idxn = 0;
						}
						if( [undefined,'cpe_operation','CONNECTION_STATUS','SERIAL_NUMBER','CPE_NAME'].includes(m.property) ) {
							idxm = 0;
						}
	
						return idxn - idxm;
					});
	
					cpevm.$refs.list.$refs.ctableInner.store.updateColumns();
					
					var ctner = document.querySelector('#cpeInfo');

                    ctner.style.width = '99.9%';
					setTimeout(function(){
						ctner.style.width = '100%';
					},1000);
				}).catch(function(){});

				vm.closeConfig();
			},
			configTable() {
				var vm = this,
                    columns = cpevm.getAllColumns();

				var columns = columns.filter(function(item){
					return vm.showCols.includes(item.prop) || ['cpe_operation','CONNECTION_STATUS','SERIAL_NUMBER','CPE_NAME'].includes(item.prop);
				});

				if(isBatchable()) {
					columns.unshift({field:'ck',checkbox: true});
				}

				cpevm.showProps = vm.showCols;
			},
			init(){
				var vm = this;
				axios.post("${ctx}/cell/CPE/getCpeSelectFilter.action",stringify({
					"operator_codes":vm.operator_code
				})).then(function(response){
					var data = response.data;
					var arr = [];
					data.map(function(item){
						if(item.value){
							arr.push({label:item.text,value:item.value})
						}
					})
					vm.advancedQueryItemList.map((items)=>{
						if('software_version' == items.value){
							items.options = arr
						}
					})
					
				})
				axios.post("${ctx}/cell/CPE/getCpeSelectFilter.action",stringify({
					"selectType":"product_model",
					"operator_codes":vm.operator_code
				})).then(function(response){
					var data = response.data;
					var arr = [];
					if(data){
						data.map(function(item){
							if(item.value){
								arr.push({label:item.text,value:item.value})
							}
						})
					}
					vm.advancedQueryItemList.map((items)=>{
						if('product_model' == items.value){
							items.options = arr
						}
					})
					
					
				})
				axios.post("${ctx}/cell/CPE/getCpeSelectFilter.action",stringify({
					"selectType":"cpeMonitorModule",
					"operator_codes":vm.operator_code
				})).then(function(response){
					var data = response.data;
					var arr = [{label:'<%=rb.getString("QuanBu")%>',value:''}];
					if(data){
						data.map(function(item){
							if(item.value){
								arr.push({label:item.text,value:item.value})
							}
						})
					}
					vm.advancedQueryItemList.map((items)=>{
						if('cpeMonitorModule' == items.value){
							items.options = arr
						}
					})
				})
				axios.post("${ctx}/cell/CPE/getDeviceGroup4Combobox.action",stringify({
					"operator_code":vm.operator_code
				})).then(function(response){
					var data = response.data;
					var arr = [];
					if(data){
						data.map(function(item){
							if(item.value){
								arr.push({label:item.text,value:item.value})
							}
						})
					}
					vm.advancedQueryItemList.map((items)=>{
						if('group_id' == items.value){
							items.options = arr
						}
					})
				})
				
				
			},
            columnStatus() {
                var vm = this;

                $('#showHideItemCpe').slideDown();
            }
		},
		created() {
			var vm = this,
				map = {
					device: vm.deviceCol.map(function(item){ return item.code;}),
					lte: vm.lteCol.map(function(item){ return item.code;}),
					nr:vm.nrCol.map(function(item){ return item.code;}),
					lan: vm.lanCol.map(function(item){ return item.code;}),
					location: vm.locationCol.map(function(item){ return item.code;})
				},
				dragCodes = vm.columns.map(function(item){ return item.field}),
				showCols = cpevm.showProps;
			
			showCols.map(function(col){
				['device','lte','nr','lan','location'].map(function(code){
					if(map[code].includes(col) && !vm.form[code].includes(col)) vm.form[code].push(col);
				});
				// 初始化默认选中项
				if(dragCodes.includes(col) && !vm.dragCol.includes(col)) vm.dragCol.push(col);
			});

			vm.$nextTick(function(){
				vm.configTable();
			});
			
			// 排序
			var vm = this,
				sortCodes = cpevm.sortColumns,
				sortList = vm.columns;

			vm.columns = sortList.sort(function(n, m){
				var idxn = sortCodes.indexOf(n.field)==-1?100:sortCodes.indexOf(n.field),
					idxm = sortCodes.indexOf(m.field)==-1?100:sortCodes.indexOf(m.field);

				return idxn - idxm;
			});
		},
		mounted() {
			this.init();
			this.queryLatestInfo();
			this.columnConfig();
			closeLoading();
		}
	});
</script>