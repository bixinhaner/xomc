<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
	#viewTracePage .el-icon-time{
		line-height:1;
	}
	.ruContent .el-input{
		width:400px;
	}
	.ruContent .el-input .el-input__inner{
		width:400px;
	}
	.timeItem .el-input__inner{
		width:220px;
	}
	.ml45{
		margin-left: 66px
	}
	.mt16{
		margin-top: 16px
	}
	.ml25{
		margin-left: 25px
	}
	.taskInput{
		width:300px;
		height:30px;
		line-height:30px;	
	}
	.taskTypeBox{
		margin-top:20px;
	}
	.taskTypeBox label{
		display:block;
	}
	#viewTracePage .radioBox{
		border:1px solid #D1ECF5 !important;
		width:660px;
		padding:15px 0px 15px 0px;
	}
	#viewTracePage .el-form-item__label{
		line-height:26px;
		width:140px;
		text-align:left;
	}
	#viewTracePage .modeItem .el-radio{
		display:inline-block;
		margin-left:0px;
	} 
	#viewTracePage .modeItem{
		margin-top:30px;
	}
	.alarmBottomLine{
		background-color:#E9E9E9;
		width: 100%;
		height: 1px;
		margin-bottom: 30px; 
	}
	.titleStyML{
		margin-left: 40px;
	}
	#viewTracePage .editButton{
		padding:0 10px;
		height:24px;
		background:#F2F9FF;
		border-radius:2px;
		line-height:24px;
		cursor:pointer;
		margin-left:10px;
		border:1px solid #1DA3FC;
		display:inline-block;
		float:right;
		position:absolute;
		right:120px;
		top:-6px;
	}
	#viewTracePage .editButton i{
		font-size:14px !important;
	}
	#viewTracePage .editButton span{
		font-size:12px;
	}
	.dialogStyle .el-icon-circle-info:before{
		color:#CFCFCF;
	}
	#viewTracePage .el-pairgrid-title{
		top:-6px;
	}
	#viewTracePage .deviceItem{
		display:inline-block;
		margin-right:60px;
	}
	#viewTracePage .pairgrid-right .el-ctable-toolbar{
		padding: 10px!important;
	}
	.commonFlex { display: flex; }
	.commonDisplay { display: inline-block; }
	.commonDisplayBlock { display: block; }
	.commonBorder { border: 1px solid #E9E9E9; }
	.commonBackground { background: #F5F7FE; }
	.commonIconStyle { margin: 10px 15px; width: 30px; height: 30px; background: #E4F1FF; border-radius: 100px; line-height: 30px !important; }
	.commonIconColor:before { color: #7A7992; }
	.commonFontSize14, .commonToolBarBox .el-query .advanceQuery .el-icon { font-size: 14px; }
	.commonFontSize12 { font-size: 12px; }
	.commonColor, .commonToolBarBox .el-query .advanceQuery .el-icon:before, .commonIcon:before, .informationWarp .el-card__header { color: #7A7992; }
	.commonColor2 {  color: #999999; }
	.commonColor3 {  color: #333333; }
	.commonFontWeight { font-weight: bold; }
	.commonRight30 { margin-right: 30px; }
	.commonBorderRadius { border-radius: 10px;}
	.commonToolBarBox { height: 30px; }
	.commonBottom10 { margin-bottom: 10px; }
	.commonRight10 { padding-right: 10px;}
	.commonLeft20 { padding-left: 20px;}
	.marginLeft5 { margin-left: 5px; }
	.commonContent { justify-content: space-between; }
	.width516 { width: 516px; }
	.el-form-item { margin-bottom: 22px; }
	.el-form-item__label { color: #666666; } 
	.el-radio.is-bordered { max-width: 200px; height: 30px; padding: 7px 12px; }
	.el-checkbox.is-bordered { max-width: 200px; height: 30px; padding: 6px 12px; }
	.el-radio-group .el-radio__label, .el-checkbox-group .el-checkbox__label { font-size: 12px; line-height: unset;}
	.el-input__inner { height: 30px; }
	.selected-status:before { color: #67D972 !important; }
	.textareStyle .el-textarea__inner { resize: none; border-radius: 4px; }
	.el-input__inner { border-radius: 4px; }
</style>

<div id="viewTracePage" style="margin-top:20px;">
	<el-form :model='ruleForm' ref="ruleForm" label-position="top">
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text commonColor"><%=rb.getString("JiBenXinXi")%></span>
		</div>
		<div class='commonFlex mt16'>
			<el-form-item label='<%=rb.getString("GenZongMingCheng")%>' prop='traceName' class="ml45">
				<el-input maxlength=100  v-model="ruleForm.traceName" class="taskInput" style='width: 330px;' disabled></el-input>
				<span class='commonFontSize12 commonColor2 marginLeft5'><%=rb.getString("ChangDuXianZhi")%> 1-100</span>
			</el-form-item>
			<el-form-item label='<%=rb.getString("GenZongCanKaoHao")%>' prop='traceId' style='margin-left: 80px;'>
				<el-input v-model="ruleForm.traceId" size="mini" disabled></el-input>
			</el-form-item>
		</div> 
		<div class='commonFlex ml45' >
			<el-form-item label="<%=rb.getString("MiaoShu")%>" prop='remarks' style='margin: -5px 0 30px; '>
                <el-input v-model='ruleForm.remarks' type='textarea' :rows='3' class="width516 textareStyle" style='resize: none;' disabled></el-input>
            </el-form-item> 
      	</div>
		<div class="alarmBottomLine"></div>
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text commonColor"><%=rb.getString("GenZongPeiZhi")%></span>
		</div>
		<div class='commonFlex ml45 mt16' v-if='false'>
			<el-form-item prop="product_type" label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>">
				<el-select v-model='product_type' style='width: 230px;'>	
					<el-option v-for='item in productDataList' :label="item.name" :value="item.value"></el-option>
				</el-select>
			</el-form-item>
		</div>	
        <el-form-item label='<%=rb.getString("GenZongSheBei")%>' prop='traceDevice' class="ml45" style='margin-top: 10px;'>
			<el-input maxlength=100  v-model="ruleForm.traceDevice" class="taskInput" style='width: 330px;' disabled></el-input>
		</el-form-item>	
		
		<div class='commonFlex ml45 mt16'>
			<el-form-item label="<%=rb.getString("JieKouLeiXing") %>" prop="NEInterface" style='margin-bottom: 30px;' class='commonFlex'>
                <el-checkbox-group v-model="ruleForm.NEInterface" disabled>
                    <el-checkbox label="S1" border>S1 <%=rb.getString("HE")%> X2</el-checkbox>
                    <el-checkbox label="Uu" border style='margin-left: 20px;'>Uu</el-checkbox>
                </el-checkbox-group>
            </el-form-item>
		</div>
		<div class="alarmBottomLine"></div>
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text commonColor"><%=rb.getString("ZhiXingFangShi")%></span>
		</div>
		<div class='commonFlex mt16'>
			<el-form-item label=" " prop="executeMode" class="ml45">
	            <el-radio-group v-model="ruleForm.executeMode" disabled>
	                <el-radio label="1" border><%=rb.getString("LiJiZhiXing") %></el-radio>
	                <el-radio label="2" border style='margin-left: 20px;'><%=rb.getString("GuaQi") %></el-radio>
	                <el-radio label="3" border style='margin-left: 20px;'><%=rb.getString("DingShiZhiXing") %></el-radio>
	            </el-radio-group>
	        </el-form-item>
			<el-form-item prop='exetime' style='display:inline-block;margin: 6px 10px 0;' class='timeItem'>
				<el-date-picker value-format="yyyy-MM-dd HH:mm:ss" v-model='ruleForm.exetime' disabled type="datetime" :picker-options="pickerOptions"></el-date-picker>	
			</el-form-item>
		</div>
		<div>
			<el-form-item label='<%=rb.getString("ChiXuShiChang")%>' prop='duration' class="ml45" style='margin-bottom: 30px;'>
				<el-input maxlength=50  v-model="ruleForm.duration" size="mini" class="taskInput" disabled></el-input>
				<span class='commonFontSize12 commonColor2 marginLeft5'><%=rb.getString("ZhengXing")%>[1:30],<%=rb.getString("DanWei")%>:<%=rb.getString("FenZhongDaXie")%></span>
			</el-form-item>
		</div>
	</el-form>
</div>

<script type="text/javascript">
var traceEnbJson = '${signalingTraceProperties}';
var viewTraceVue = new Vue({
	el:'#viewTracePage',
	data(){
		return {
			ruleForm:{
				traceName: '',
				traceId: '',
				remarks: '',
				NEInterface: [],
				traceDevice:'',
				executeMode:'',
				exetime:'',
				duration:''
			},
			pickerOptions:{
				disabledDate(time){
					return time.getTime()< Date.now()-8.64e7;
				}
			},
			productDataList:[],
			product_type:'',
			readonly: false
		}
	},
	watch:{
	},
	methods:{ 
		init(type, row){
			var vm = this;
			vm.readonly = type == 'readonly';

			var traceEnbObj = JSON.parse(traceEnbJson);
			vm.ruleForm.traceName= traceEnbObj.trace_name;
			vm.ruleForm.traceId= traceEnbObj.trace_id;
			vm.ruleForm.remarks= traceEnbObj.remarks;
			//选中的产品类型字段是否回显
			vm.ruleForm.traceDevice = traceEnbObj.trace_device.replace('(', '').replace(')', '');
			vm.defaultChecked = [vm.ruleForm.traceDevice+''];
			vm.ruleForm.NEInterface= traceEnbObj.ne_interface.split(',');
			vm.ruleForm.executeMode= traceEnbObj.execute_mode;
			if(traceEnbObj.execute_mode == '3'){
				vm.ruleForm.exetime= traceEnbObj.start_time;
			}else{
				vm.ruleForm.exetime= '';
			}
			vm.ruleForm.duration= traceEnbObj.duration;
		}
	},
	
	mounted(){
		eventBus.$off('init-config').$on('init-config', this.init); 
	}
})
</script>
