<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<div id="TRX_POWER1" style="width: 1100px;height:500px;"></div>  
<div id="TRX_POWER2" style="width: 1100px;height:500px;"></div> 

<script type="text/javascript">

var sn ="${sn}";
var CellCode = "${cellCode}";
var Xdata = "${TRACE_TIME_DATA}".split(",");
var trxpower1Data = parseFloatData("${TRX_POWER1_DATA}".split(","));
var trxpower2Data = parseFloatData("${TRX_POWER2_DATA}".split(",")); 
generateTrxCharts("TRX_POWER1","TRX_POWER1",Xdata,trxpower1Data,"dbm");
generateTrxCharts("TRX_POWER2","TRX_POWER2",Xdata,trxpower2Data,"dbm");
 

function generateTrxCharts(elemId,title,Xdata,Ydata,yFormatUnit){
	var myChart = echarts.init(document.getElementById(elemId));   
			 var option = {  
					    title : {  
					        text: title  
					    },  
					    tooltip : {  
					        trigger: 'axis'  
					    }, 
					    color:['#48cda6'],
					    legend: {  
					    	x: 'center',
					    	y: 'top',
					        data:[sn]
					       /*  textStyle:{
					        	color: '#333',
					        	fontSize: 12
					        } */
					    },  
					    toolbox: {  
					        show : true,  
					        feature : {  
					        	/*mark:{show :true},
					        	 dataView: {show :true,readOnly:false},
					        	magicType:{show:true,type:['line','bar']},
					        	restore:{show:true},*/
					            saveAsImage : {show: false}  
					        }  
					    },  
					    calculable : true,
					    xAxis : [  
					        {  
					            type : 'category',  
					            boundaryGap : false, 
					            data : Xdata
					        }  
					    ],  
					    yAxis : [  
					        {  
					            type : 'value',  
					            axisLabel : {  
					                formatter: '{value} ' + yFormatUnit  
					            }  
					        }  
					    ],  
					    series : [  
						     {  
						            name:title,  
						            type:'line',  
						            data:Ydata,  
						            markPoint : {  
						            	 data : [  
								                    {type : 'min', name : 'Min'},  
								                    {type : 'max', name : 'Max'}  
								                ]  
						            },  
						            markLine : {  
						                data : [  
						                    {type : 'average', name : 'Avg'} 
						                ]  
						            }
						            
						     }  
					    ]  
				}; 
            // 为echarts对象加载数据   
            myChart.setOption(option);  
  }
  
  function parseFloatData(arr){
	  var retArr = [];
	  for(var i =0;i<arr.length;i++){
		  retArr[i] = parseFloat(arr[i]);
	  }
	  
	  return retArr;
  }
	  
</script>

